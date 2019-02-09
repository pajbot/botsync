package main

import "errors"

var (
	errUnhandledTopic = errors.New("handleTopic: unhandled topic")
)
