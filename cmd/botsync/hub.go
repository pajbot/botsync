package main

import (
	"encoding/json"
	"fmt"

	"github.com/pajlada/botsync/pkg/protocol"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	process chan *SourceMessage

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	subscriptions map[string][]*Client
}

func newHub() *Hub {
	return &Hub{
		process:    make(chan *SourceMessage),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),

		subscriptions: make(map[string][]*Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			for topic := range client.subscriptions {
				for i, topicSubscription := range h.subscriptions[topic] {
					if topicSubscription == client {
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
				data := &protocol.SubscribeMessageData{}
				err := json.Unmarshal(message.Data, data)
				if err != nil {
					fmt.Println("err:", err)
					continue
				}
				h.subscribe(message.source, data.Topic)

			case "PUBLISH":
				data := &protocol.PublishMessageData{}
				err := json.Unmarshal(message.Data, data)
				if err != nil {
					fmt.Println("err:", err)
					continue
				}
				h.publish(message.source, data.Topic, data.Payload)

			default:
				fmt.Println("Received message to process:", message)
			}

		case message := <-h.broadcast:
			fmt.Println("Received message:", string(message))
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) subscribe(client *Client, topic string) {
	if _, ok := client.subscriptions[topic]; ok {
		return
	}
	fmt.Println("Subscribed to", topic)

	client.subscriptions[topic] = true
	h.subscriptions[topic] = append(h.subscriptions[topic], client)
}

func (h *Hub) publish(client *Client, topic string, data interface{}) {
	listeners, ok := h.subscriptions[topic]
	if !ok {
		return
	}

	for _, listener := range listeners {
		// TODO: check error
		listener.send <- []byte("xd")
	}
}
