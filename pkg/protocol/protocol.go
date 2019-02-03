package protocol

import "encoding/json"

// Valid types:
// "SUBSCRIBE"
// "PUBLISH"

type BaseMessage struct {
	Type string

	Data json.RawMessage
}

type SubscribeMessageData struct {
	Topic string
}

type SubscribeMessage struct {
	Type string
	Data SubscribeMessageData
}

type PublishMessageData struct {
	Topic   string
	Payload string
}

type PublishMessage struct {
	Type string
	Data PublishMessageData
}
