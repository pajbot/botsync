package client

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pajlada/botsync/pkg/protocol"
)

var (
	ErrDisconnected = errors.New("client disconnected")
)

type Client struct {
	host string
	conn *websocket.Conn

	disconnect chan bool
	send       chan interface{}

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

func (c *Client) OnMessage(cb func(message *protocol.Message)) {
	c.onMessage = cb
}

func (c *Client) OnConnect(cb func()) {
	c.onConnect = cb
}

func (c *Client) Connect() error {
	var err error

	done := make(chan struct{})

	c.conn, _, err = websocket.DefaultDialer.Dial(c.host, nil)
	if err != nil {
		return err
	}
	defer c.conn.Close()

	if c.onConnect != nil {
		c.onConnect()
	}

	// Reader
	go func() {
		defer close(done)
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				return
			}

			parsedMessage := &protocol.Message{}
			err = json.Unmarshal(message, parsedMessage)
			if err != nil {
				log.Println("read parse error:", err)
				continue
			}
			if c.onMessage != nil {
				c.onMessage(parsedMessage)
			}
		}
	}()

	for {
		select {
		case <-done:
			return errors.New("xd")
		case msg := <-c.send:
			payload, err := json.Marshal(msg)
			err = c.conn.WriteMessage(websocket.TextMessage, payload)
			if err != nil {
				log.Println("write:", err)
				return err
			}
		case <-c.disconnect:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
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
		log.Println("unable to disconnect")
	}
}
