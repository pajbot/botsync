package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pajlada/botsync/internal/config"
	"github.com/pajlada/stupidmigration"
)

type BotSyncServer struct {
	Host          string
	WebsocketPath string

	afkDatabase AFKDatabase

	hub *Hub
	db  *sql.DB
}

func New(db *sql.DB) (s *BotSyncServer, err error) {
	s = &BotSyncServer{
		Host:          "0.0.0.0:8080",
		WebsocketPath: "/ws/pubsub",

		hub: newHub(),
		db:  db,
	}

	err = s.db.Ping()
	if err != nil {
		return
	}

	s.db.SetMaxOpenConns(10)

	err = stupidmigration.Migrate(config.GetMigrationsPath(), db)
	if err != nil {
		return
	}

	s.afkDatabase = AFKDatabase{
		hub:   s.hub,
		users: make(map[string]*afkUser),
	}

	s.hub.publishHandlers["ping"] = handlePing
	s.hub.publishHandlers["afk.set"] = s.afkDatabase.setAFK
	s.hub.publishHandlers["back.set"] = s.afkDatabase.setBack
	s.hub.subscribeHandlers["afk"] = s.afkDatabase.subscribeAFK

	return
}

func (s *BotSyncServer) Listen() error {
	go s.hub.run()

	http.HandleFunc(s.WebsocketPath, func(w http.ResponseWriter, r *http.Request) {
		serveWs(s.hub, w, r, s.db)
	})

	return http.ListenAndServe(s.Host, nil)
}

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
