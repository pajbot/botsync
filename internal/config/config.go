package config

import (
	"os"
)

func GetMigrationsPath() string {
	const e = `BOTSYNC_MIGRATIONS_PATH`
	const defaultValue = `../../migrations`

	if value, ok := os.LookupEnv(e); ok {
		return value
	}

	return defaultValue
}

func GetHost() string {
	const e = `BOTSYNC_HOST`
	const defaultValue = `:8080`

	if value, ok := os.LookupEnv(e); ok {
		return value
	}

	return defaultValue
}

func GetWebsocketPath() string {
	const e = `BOTSYNC_WS_PATH`
	const defaultValue = `/ws/pubsub`

	if value, ok := os.LookupEnv(e); ok {
		return value
	}

	return defaultValue
}

func GetDSN() string {
	const e = `BOTSYNC_DSN`
	const defaultValue = `postgres:///botsync?sslmode=disable`

	if value, ok := os.LookupEnv(e); ok {
		return value
	}

	return defaultValue
}
