package protocol

import "encoding/json"

// Valid types:
// "SUBSCRIBE"
// "PUBLISH"

type Message struct {
	Type  string
	Topic string

	Historic bool

	Data json.RawMessage `json:",omitempty"`
}

type OutgoingMessage struct {
	Type  string
	Topic string

	Historic bool

	Data interface{} `json:",omitempty"`
}

func NewSubscribeMessage(topic string, parameters interface{}) *OutgoingMessage {
	return &OutgoingMessage{
		Type:  "SUBSCRIBE",
		Topic: topic,

		Data: parameters,
	}
}

func NewPingMessage() *Message {
	return &Message{
		Type:  "PUBLISH",
		Topic: "ping",
	}
}

func NewPongMessage() *Message {
	return &Message{
		Type:  "PUBLISH",
		Topic: "pong",
	}
}

// prebaked messages
var (
	PingBytes []byte
	PongBytes []byte
)

func init() {
	var err error

	PingBytes, err = json.Marshal(NewPingMessage())
	if err != nil {
		panic(err)
	}

	PongBytes, err = json.Marshal(NewPongMessage())
	if err != nil {
		panic(err)
	}
}
