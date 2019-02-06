package protocol

import "encoding/json"

// Valid types:
// "SUBSCRIBE"
// "PUBLISH"

type BaseMessage struct {
	Type  string
	Topic string

	Data json.RawMessage
}

type SubscribeMessage struct {
	Type  string
	Topic string
}

type PublishMessageData struct {
	Payload string
}

type PublishMessage struct {
	Type  string
	Topic string

	Data PublishMessageData
}

// PongMessage is a response to { "Type": "Publish", "Topic": "PING" }
type PongMessage struct {
	Type  string // always PUBLISH
	Topic string // always PONG

	Data struct {
		Message string
	}
}
