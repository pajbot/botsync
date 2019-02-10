package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/pajlada/botsync/pkg/protocol"
)

type SubscriptionParameter interface {
	Matches(interface{}) bool
}

type Subscription struct {
	client *Client

	parameters SubscriptionParameter
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

	publishHandlers map[string]func(client *Client, unparsedData json.RawMessage) error
}

func newHub() *Hub {
	return &Hub{
		process:    make(chan *SourceMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),

		subscriptions: make(map[string][]*Subscription),

		publishHandlers: make(map[string]func(client *Client, unparsedData json.RawMessage) error),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
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
	defer h.subscriptionsMutex.Unlock()
	h.subscriptions[topic] = append(h.subscriptions[topic], &Subscription{
		client:     client,
		parameters: parameters,
	})
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
	}
	h.subscribe(message.source, message.Topic, parameters)

	return nil
}

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
