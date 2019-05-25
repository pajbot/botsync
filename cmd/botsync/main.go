package main

import (
	"github.com/pajbot/botsync/internal/dbhelp"
	"github.com/pajbot/botsync/pkg/server"
)

func main() {
	db, err := dbhelp.Connect()
	if err != nil {
		return
	}

	server, err := server.New(db)
	if err != nil {
		panic(err)
	}
	err = server.Listen()
	if err != nil {
		panic(err)
	}
}
