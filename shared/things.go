package shared

import "database/sql"

type Config struct {
	SlackToken string
	Port       string
	UserDB     *sql.DB
}
