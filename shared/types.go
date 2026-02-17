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
	Departure    int64  `db:"departure" json:"departure"`
}

type FlightState struct {
	FlightID         string `db:"flight_id"`
	Status           string `db:"status"`
	OriginGate       string `db:"origin_gate"`
	OriginTerminal   string `db:"origin_terminal"`
	DestGate         string `db:"dest_gate"`
	DestTerminal     string `db:"dest_terminal"`
	DepScheduled     int64  `db:"dep_scheduled"`
	DepEstimated     int64  `db:"dep_estimated"`
	DepActual        int64  `db:"dep_actual"`
	TakeOffActual    int64  `db:"takeoff_actual"`
	TakeOffEstimated int64  `db:"takeoff_estimated"`
	LandingActual    int64  `db:"landing_actual"`
	LandingEstimated int64  `db:"landing_estimated"`
	ArrScheduled     int64  `db:"arr_scheduled"`
	ArrEstimated     int64  `db:"arr_estimated"`
	ArrActual        int64  `db:"arr_actual"`
	Altitude         int    `db:"altitude"`
	Groundspeed      int    `db:"groundspeed"`
	UpdatedAt        int64  `db:"updated_at"`
}
