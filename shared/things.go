package shared

import (
	"database/sql"

	"github.com/slack-go/slack"
)

type Config struct {
	Port        string
	UserDB      *sql.DB
	SlackClient *slack.Client
}

type Command struct {
	Name        string
	Description string
	Usage       string
	Execute     func(slashCommand slack.SlashCommand, config Config) (blocks []slack.Block, inChannel bool, after func() error)
}
