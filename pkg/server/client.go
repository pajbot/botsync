package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pajbot/botsync/pkg/protocol"
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	subscriptions map[string]bool

	twitchUserID string
}

func newClient(hub *Hub, conn *websocket.Conn) *Client {
	client := &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}

	return client
}

func (c *Client) start() {
	c.hub.register <- c

	c.send <- protocol.AuthenticationSuccessBytes

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go c.writePump()
	go c.readPump()
}

func (c *Client) authenticate(db *sql.DB) chan bool {
	done := make(chan bool)

	go func() {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			done <- false
			return
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		var authentication protocol.Authentication

		err = json.Unmarshal(message, &authentication)
		if err != nil {
			done <- false
			return
		}

		c.twitchUserID = authentication.TwitchUserID

		if c.twitchUserID == "" {
			done <- false
			return
		}

		const query = `SELECT authentication_token FROM client WHERE twitch_user_id=$1`
		row := db.QueryRow(query, c.twitchUserID)
		var authToken string
		err = row.Scan(&authToken)
		if err != nil {
			if err != sql.ErrNoRows {
				fmt.Println("Error during row scan:", err)
			}
			done <- false
			return
		}
		done <- authentication.AuthenticationToken == authToken
	}()

	return done
}

func (c *Client) disconnect() {
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	c.conn.Close()
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		// Parse to JSON
		parsedMessage := &SourceMessage{
			source: c,
		}

		err = json.Unmarshal(message, parsedMessage)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		c.hub.process <- parsedMessage
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
