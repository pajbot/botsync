package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/lib/pq"
	"github.com/pajbot/botsync/internal/dbhelp"
	"github.com/pajbot/utils"
)

func generateAuthToken() (string, error) {
	return utils.GenerateRandomString(32)
}

func setToken(db *sql.DB, twitchUserID string, updateOnConflict bool) error {
	if twitchUserID == "" {
		return fmt.Errorf("missing user id. Usage: %s %s TWITCH_USER_ID", os.Args[0], flag.Arg(0))
	}

	authToken, err := generateAuthToken()
	if err != nil {
		return fmt.Errorf("error generating auth token: %s", err)
	}

	query := `INSERT INTO client (twitch_user_id, authentication_token) VALUES ($1, $2)`
	if updateOnConflict {
		query = `INSERT INTO client (twitch_user_id, authentication_token) VALUES ($1, $2) ON CONFLICT(twitch_user_id) DO UPDATE SET authentication_token = $2`
	}
	_, err = db.Exec(query, twitchUserID, authToken)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation error code
				return errors.New("this user already has a token, use regenerate instead of generate")
			}
		}
		return err
	}

	fmt.Println("Authentication token for", twitchUserID, "is:", authToken)

	return nil
}

func main() {
	flag.Parse()
	db, err := dbhelp.Connect()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	switch flag.Arg(0) {
	case "regenerate":
		err := setToken(db, flag.Arg(1), true)
		if err != nil {
			fmt.Println("error regenerating token:", err)
			os.Exit(1)
		}

	case "generate":
		err := setToken(db, flag.Arg(1), false)
		if err != nil {
			fmt.Println("error regenerating token:", err)
			os.Exit(1)
		}

	default:
		fmt.Println("XD")
	}
}
