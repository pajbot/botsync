package main

import (
	"github.com/pajlada/botsync/internal/dbhelp"
	"github.com/pajlada/botsync/pkg/server"
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
