package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/pajlada/botsync/pkg/protocol"
	"github.com/pajlada/pajbot2/pkg/utils"
)

type SubscriptionParameter interface {
	Matches(interface{}) bool
}

type Subscription struct {
	client *Client

	parameters SubscriptionParameter
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type SourceMessage struct {
	protocol.Message

	source *Client
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	process chan *SourceMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	subscriptionsMutex sync.Mutex
	subscriptions      map[string][]*Subscription

	publishHandlers   map[string]func(client *Client, unparsedData json.RawMessage) error
	subscribeHandlers map[string]func(client *Client, parameters SubscriptionParameter) error
}

func newHub() *Hub {
	return &Hub{
		process:    make(chan *SourceMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),

		subscriptions: make(map[string][]*Subscription),

		publishHandlers:   make(map[string]func(client *Client, unparsedData json.RawMessage) error),
		subscribeHandlers: make(map[string]func(client *Client, parameters SubscriptionParameter) error),
	}
}

func (h *Hub) unregisterClient(client *Client) {
	h.subscriptionsMutex.Lock()
	defer h.subscriptionsMutex.Unlock()

	for topic := range client.subscriptions {
		for i, topicSubscription := range h.subscriptions[topic] {
			if topicSubscription.client == client {
				copy(h.subscriptions[topic][i:], h.subscriptions[topic][i+1:])
				h.subscriptions[topic][len(h.subscriptions[topic])-1] = nil // or the zero vh.subscriptions[topic]lue of T
				h.subscriptions[topic] = h.subscriptions[topic][:len(h.subscriptions[topic])-1]
			}
		}
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			h.unregisterClient(client)

			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case message := <-h.process:
			switch message.Type {
			case "SUBSCRIBE":
				err := h.handleSubscribe(message)
				if err != nil {
					fmt.Println("Error handling subscription:", err)
				}

			case "PUBLISH":
				h.handlePublish(message.source, message.Topic, message.Data)
			}
		}
	}
}

func (h *Hub) subscribe(client *Client, topic string, parameters SubscriptionParameter) {
	// TODO: Ensure the client is not subscribed with the same parameters already
	if _, ok := client.subscriptions[topic]; ok {
		return
	}

	client.subscriptions[topic] = true

	h.subscriptionsMutex.Lock()
	h.subscriptions[topic] = append(h.subscriptions[topic], &Subscription{
		client:     client,
		parameters: parameters,
	})
	h.subscriptionsMutex.Unlock()

	if cb, ok := h.subscribeHandlers[topic]; ok {
		go cb(client, parameters)
	}

}

func (h *Hub) handleSubscribe(message *SourceMessage) error {
	var parameters SubscriptionParameter

	switch message.Topic {
	case "afk":
		messageParameters := &protocol.AFKSubscribeParameters{}
		err := json.Unmarshal(message.Data, messageParameters)
		if err != nil {
			return err
		}
		parameters = messageParameters
	case "back":
		messageParameters := &protocol.BackSubscribeParameters{}
		err := json.Unmarshal(message.Data, messageParameters)
		if err != nil {
			return err
		}
		parameters = messageParameters
	default:
		return fmt.Errorf("unknown topic %s", message.Topic)
	}

	h.subscribe(message.source, message.Topic, parameters)

	return nil
}

var (
	errUnhandledTopic = errors.New("handleTopic: unhandled topic")
)

func (h *Hub) handlePublish(client *Client, topic string, unparsedData json.RawMessage) error {
	if cb, ok := h.publishHandlers[topic]; ok {
		return cb(client, unparsedData)
	}

	log.Println("Unhandled topic", topic)
	return errUnhandledTopic
}

func (h *Hub) publish(message *protocol.OutgoingMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.subscriptionsMutex.Lock()
	defer h.subscriptionsMutex.Unlock()
	if subscriptions, ok := h.subscriptions[message.Topic]; ok {
		for _, subscription := range subscriptions {
			if subscription.parameters != nil {
				if !subscription.parameters.Matches(message.Data) {
					continue
				}
			}
			subscription.client.send <- payload
		}
	}

	return nil
}

func handlePing(client *Client, unparsedData json.RawMessage) error {
	client.send <- protocol.PongBytes
	return nil
}

type afkUser struct {
	parameters *protocol.AFKParameters
	time       time.Time
}

type AFKDatabase struct {
	hub *Hub

	usersMutex sync.Mutex
	users      map[string]*afkUser
}

func (d *AFKDatabase) setUserAFK(parameters *protocol.AFKParameters) bool {
	d.usersMutex.Lock()
	defer d.usersMutex.Unlock()
	if _, ok := d.users[parameters.UserID]; ok {
		fmt.Println("User is already afk")
		// User is already AFK
		return false
	}

	d.users[parameters.UserID] = &afkUser{
		parameters: parameters,
		time:       time.Now(),
	}

	return true
}

func (d *AFKDatabase) setAFK(client *Client, unparsedData json.RawMessage) error {
	var parameters protocol.AFKParameters
	_ = json.Unmarshal(unparsedData, &parameters)

	if !d.setUserAFK(&parameters) {
		return nil
	}

	outboundMessage := &protocol.OutgoingMessage{
		Type:  "PUBLISH",
		Topic: "afk",
		Data:  parameters,
	}

	go d.hub.publish(outboundMessage)

	return nil
}

func (d *AFKDatabase) subscribeAFK(client *Client, subscriptionParameters SubscriptionParameter) error {
	d.usersMutex.Lock()
	defer d.usersMutex.Unlock()
	for _, afkUser := range d.users {
		outboundMessage := &protocol.OutgoingMessage{
			Type:     "PUBLISH",
			Topic:    "afk",
			Historic: true,
			Data:     afkUser.parameters,
		}

		bytes, _ := json.Marshal(outboundMessage)

		client.send <- bytes
	}

	return nil
}

func (d *AFKDatabase) setBack(client *Client, unparsedData json.RawMessage) error {
	d.usersMutex.Lock()
	defer d.usersMutex.Unlock()

	var setParameters protocol.BackSetParameters
	_ = json.Unmarshal(unparsedData, &setParameters)

	afkUser, ok := d.users[setParameters.UserID]
	if !ok {
		// User was not AFK to begin with
		return nil
	}
	delete(d.users, setParameters.UserID)

	afkParameters := afkUser.parameters

	afkTimeInMilliseconds := utils.MaxInt64(0, int64(math.Ceil(float64(time.Since(afkUser.time).Nanoseconds())/1e6)))

	parameters := protocol.BackParameters{
		UserID:   setParameters.UserID,
		UserName: setParameters.UserName,

		ChannelID:   setParameters.ChannelID,
		ChannelName: setParameters.ChannelName,

		Reason: afkParameters.Reason,

		AFKChannelID:   afkParameters.ChannelID,
		AFKChannelName: afkParameters.ChannelName,

		Duration: afkTimeInMilliseconds,
	}

	outboundMessage := &protocol.OutgoingMessage{
		Type:  "PUBLISH",
		Topic: "back",
		Data:  parameters,
	}
	go d.hub.publish(outboundMessage)

	return nil
}
