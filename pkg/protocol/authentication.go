package protocol

import "encoding/json"

type Authentication struct {
	TwitchUserID        string
	AuthenticationToken string
}

type AuthenticationResponse struct {
	Success bool
	Message string
}

// prebaked messages
var (
	AuthenticationSuccessBytes []byte
)

func init() {
	var err error

	AuthenticationSuccessBytes, err = json.Marshal(&AuthenticationResponse{
		Success: true,
		Message: "success",
	})

	if err != nil {
		panic(err)
	}
}
