package shared

import (
	"database/sql"
	"flight-tracker-slack/maps"
	"time"

	"github.com/slack-go/slack"
)

type Config struct {
	Port          string
	UserDB        *sql.DB
	SlackClient   *slack.Client
	TileStore     *maps.TileStore
	SigningSecret string
	SlackToken    string
}

type Command struct {
	Name        string
	Description string
	Usage       string
	Execute     func(slashCommand slack.SlashCommand, config Config) (blocks []slack.Block, inChannel bool, after func() error)
}

type NotificationLog struct {
	FlightID  string
	AlertType string
	SentAt    time.Time
}

type Interaction struct {
	Prefix  string
	Execute func(callback slack.InteractionCallback, config Config)
}

type Flight struct {
	ID           string `db:"id" json:"id"`
	FlightNumber string `db:"flight_number" json:"flight_number"`
	SlackChannel string `db:"slack_channel" json:"slack_channel"`
	SlackUserID  string `db:"slack_user_id" json:"slack_user_id"`
	Departure    string `db:"departure" json:"departure"`
}
