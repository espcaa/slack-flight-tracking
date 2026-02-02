package shared

import (
	"database/sql"

	"github.com/slack-go/slack"
)

type Config struct {
	SlackToken string
	Port       string
	UserDB     *sql.DB
}

type Command struct {
	Name        string
	Description string
	Usage       string
	Execute     func(commandText string, responseURL string, config Config) []slack.Block
}
