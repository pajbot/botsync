package protocol

type BackSetParameters struct {
	UserID   string
	UserName string

	// Channel the user just came back in
	ChannelID   string
	ChannelName string
}

type BackParameters struct {
	UserID   string
	UserName string

	// Reason the user went AFK for
	Reason string

	// Channel the user originally went AFK in
	AFKChannelID   string
	AFKChannelName string

	// Channel the user just came back in
	ChannelID   string
	ChannelName string

	// How long the user has been AFK in milliseconds
	Duration int64
}

type BackSubscribeParameters struct {
	ChannelID string
}

func (p *BackSubscribeParameters) Matches(rhsInterface interface{}) bool {
	switch rhs := rhsInterface.(type) {
	case BackSubscribeParameters:
		return rhs.ChannelID == p.ChannelID
	case BackParameters:
		return rhs.ChannelID == p.ChannelID
	}

	return false
}

func NewBackMessage(params *BackParameters) *OutgoingMessage {
	return &OutgoingMessage{
		Type:  "PUBLISH",
		Topic: "back.set",

		Data: params,
	}
}

func NewBackSubscribeMessage(channelID string) *OutgoingMessage {
	return &OutgoingMessage{
		Type:  "SUBSCRIBE",
		Topic: "back",

		Data: &BackSubscribeParameters{
			ChannelID: channelID,
		},
	}
}
