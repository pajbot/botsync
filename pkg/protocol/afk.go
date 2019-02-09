package protocol

type AFKParameters struct {
	UserID   string
	UserName string

	ChannelID   string
	ChannelName string

	Reason string
}

type AFKSubscribeParameters struct {
	ChannelID string
}

func (p *AFKSubscribeParameters) Matches(rhsInterface interface{}) bool {
	switch rhs := rhsInterface.(type) {
	case AFKSubscribeParameters:
		return rhs.ChannelID == p.ChannelID
	case AFKParameters:
		return rhs.ChannelID == p.ChannelID
	}

	return false
}

func NewAFKMessage(params *AFKParameters) *OutgoingMessage {
	return &OutgoingMessage{
		Type:  "PUBLISH",
		Topic: "afk.set",

		Data: params,
	}
}

func NewAFKSubscribeMessage(channelID string) *OutgoingMessage {
	return &OutgoingMessage{
		Type:  "SUBSCRIBE",
		Topic: "afk",

		Data: &AFKSubscribeParameters{
			ChannelID: channelID,
		},
	}
}
