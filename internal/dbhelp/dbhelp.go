package dbhelp

import (
	"database/sql"

	"github.com/pajlada/botsync/internal/config"
)

func Connect() (*sql.DB, error) {
	return sql.Open("postgres", config.GetDSN())
}
