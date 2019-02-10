package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pajlada/botsync/pkg/protocol"
	"github.com/pajlada/stupidmigration"

	_ "github.com/lib/pq"
)

type SourceMessage struct {
	protocol.Message

	source *Client
}

var host = flag.String("host", ":8080", "listen host")

func handlePing(client *Client, unparsedData json.RawMessage) error {
	client.send <- protocol.PongBytes
	return nil
}

type AFKDatabase struct {
	hub *Hub

	users map[string]*protocol.AFKParameters
}

func (d *AFKDatabase) setAFK(client *Client, unparsedData json.RawMessage) error {
	var parameters protocol.AFKParameters
	_ = json.Unmarshal(unparsedData, &parameters)

	if _, ok := d.users[parameters.UserID]; ok {
		// User is already AFK
		return nil
	}

	d.users[parameters.UserID] = &parameters

	outboundMessage := &protocol.OutgoingMessage{
		Type:  "PUBLISH",
		Topic: "afk",
		Data:  parameters,
	}

	go d.hub.publish(outboundMessage)

	return nil
}

func (d *AFKDatabase) setBack(client *Client, unparsedData json.RawMessage) error {
	var setParameters protocol.BackSetParameters
	_ = json.Unmarshal(unparsedData, &setParameters)

	afkParameters, ok := d.users[setParameters.UserID]
	if !ok {
		// User was not AFK to begin with
		return nil
	}
	delete(d.users, setParameters.UserID)

	parameters := protocol.BackParameters{
		UserID:   setParameters.UserID,
		UserName: setParameters.UserName,

		ChannelID:   setParameters.ChannelID,
		ChannelName: setParameters.ChannelName,

		Reason: afkParameters.Reason,

		AFKChannelID:   afkParameters.ChannelID,
		AFKChannelName: afkParameters.ChannelName,
	}

	outboundMessage := &protocol.OutgoingMessage{
		Type:  "PUBLISH",
		Topic: "back",
		Data:  parameters,
	}
	go d.hub.publish(outboundMessage)

	return nil
}

func main() {
	flag.Parse()
	hub := newHub()

	db, err := sql.Open("postgres", "postgres:///botsync?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxOpenConns(10)

	err = stupidmigration.Migrate("../../migrations", db)
	if err != nil {
		log.Fatal("Error running migrations:", err)
	}

	afkDatabase := AFKDatabase{
		hub:   hub,
		users: make(map[string]*protocol.AFKParameters),
	}

	hub.publishHandlers["ping"] = handlePing
	hub.publishHandlers["afk.set"] = afkDatabase.setAFK
	hub.publishHandlers["back.set"] = afkDatabase.setBack

	go hub.run()
	http.HandleFunc("/ws/pubsub", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r, db)
	})
	fmt.Println("Listening on", *host)
	err = http.ListenAndServe(*host, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(hub, conn)

	go func(client *Client) {
		done := client.authenticate(db)

		select {
		case status := <-done:
			if status {
				client.start()
				fmt.Println("["+client.conn.RemoteAddr().String()+"] success auth with id", client.twitchUserID)
				return
			}

			fmt.Println("["+client.conn.RemoteAddr().String()+"] failed auth with id", client.twitchUserID)
		case <-time.After(5 * time.Second):
			fmt.Println("timed out")
		}

		client.disconnect()
	}(client)
}
