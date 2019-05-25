// +build integration

package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/pajbot/botsync/pkg/client"
	"github.com/pajbot/botsync/pkg/server"

	_ "github.com/lib/pq"
)

const sqlHost = `postgres://postgres:penis123@localhost:5433/postgres?sslmode=disable`

func startServer() {
	db, err := sql.Open("postgres", sqlHost)
	if err != nil {
		fmt.Println("Error connecting to db:", err)
		return
	}

	server, err := server.New(db)
	if err != nil {
		fmt.Println("Error starting server:", err)
		panic(err)
	}
	fmt.Println("aaaaaa")
	err = server.Listen()
	if err != nil {
		fmt.Println("Error listening server:", err)
		panic(err)
	}
	fmt.Println("xd")
}

func waitForServer() {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws/pubsub"}
	c := make(chan bool)
	// c2 := make(chan bool)
	client := client.NewClient(u.String())
	client.OnConnect(func() {
		fmt.Println("Connected")
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		close(c)
	})

	go func() {
		for {
			client.Connect()
			select {
			case <-c:
				return
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}
	}()

	select {
	case <-c:
		client.Disconnect()
		return
	case <-time.After(3 * time.Second):
		fmt.Println("server connection timed out")
		os.Exit(1)
	}
}

func init() {
	go func() {
		startServer()
	}()
	waitForServer()
}

func TestHelloWorld(t *testing.T) {
	// <-time.After(1 * time.Second)
	// t.Fatal("not implemented")
}
