package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pajbot/botsync/pkg/protocol"
)

var (
	ErrDisconnected = errors.New("client disconnected")

	ErrAuthenticationFailed = errors.New("authentication failed")
)

type Client struct {
	host string
	conn *websocket.Conn

	disconnect chan bool
	send       chan interface{}

	authentication protocol.Authentication

	onMessage func(message *protocol.Message)
	onConnect func()
}

func NewClient(host string) *Client {
	return &Client{
		host:       host,
		disconnect: make(chan bool),
		send:       make(chan interface{}, 3),
	}
}

func (c *Client) SetAuthentication(authentication protocol.Authentication) {
	c.authentication = authentication
}

func (c *Client) OnMessage(cb func(message *protocol.Message)) {
	c.onMessage = cb
}

func (c *Client) OnConnect(cb func()) {
	c.onConnect = cb
}

func (c *Client) authenticate() error {
	payload, err := json.Marshal(c.authentication)
	if err != nil {
		return err
	}

	if err = c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		return err
	}

	return nil
}

func (c *Client) readAuthenticationResponse() error {
	_, rawMessage, err := c.conn.ReadMessage()
	if err != nil {
		return err
	}

	var msg protocol.AuthenticationResponse
	err = json.Unmarshal(rawMessage, &msg)
	if err != nil {
		return err
	}

	if !msg.Success {
		return ErrAuthenticationFailed
	}

	return nil
}

func (c *Client) Connect() error {
	var err error

	done := make(chan error)

	c.conn, _, err = websocket.DefaultDialer.Dial(c.host, nil)
	if err != nil {
		return err
	}
	defer c.conn.Close()

	c.authenticate()

	if err := c.readAuthenticationResponse(); err != nil {
		return err
	}

	if c.onConnect != nil {
		c.onConnect()
	}

	// Reader
	go func() {
		defer close(done)
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				done <- err
				return
			}

			parsedMessage := &protocol.Message{}
			err = json.Unmarshal(message, parsedMessage)
			if err != nil {
				done <- err
				continue
			}
			if c.onMessage != nil {
				c.onMessage(parsedMessage)
			}
		}
	}()

	for {
		select {
		case err := <-done:
			return err
		case msg := <-c.send:
			payload, err := json.Marshal(msg)
			err = c.conn.WriteMessage(websocket.TextMessage, payload)
			if err != nil {
				fmt.Println("write:", err)
				return err
			}
		case <-c.disconnect:
			fmt.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				fmt.Println("write close:", err)
				return err
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return ErrDisconnected
		}
	}
}

func (c *Client) Send(message interface{}) error {
	c.send <- message
	return nil
}

func (c *Client) Disconnect() {
	select {
	case c.disconnect <- true:
	default:
		fmt.Println("unable to disconnect")
	}
}
