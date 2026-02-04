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

type FlightTrack struct {
	ID                 string    // Flight Code/ID
	SlackChannel       string    // Destination for alerts
	LastStatus         string    // e.g., "scheduled", "active", "landed"
	IsAirborne         bool      // State variable for takeoff/landing logic
	ScheduledDep       time.Time // Original departure to check for delays
	CurrentETA         time.Time // Latest estimated arrival to check for shifts
	LastPeriodicUpdate time.Time // When the last 2-hour update was sent
	IsCompleted        bool      // True when flight hits the gate (stops polling)
}

type NotificationLog struct {
	FlightID  string
	AlertType string
	SentAt    time.Time
}

func MapRowToTrack(id string, channel string, air int, dep int64, eta int64, update int64, comp int) FlightTrack {
	return FlightTrack{
		ID:                 id,
		SlackChannel:       channel,
		IsAirborne:         air == 1,
		ScheduledDep:       time.Unix(dep, 0),
		CurrentETA:         time.Unix(eta, 0),
		LastPeriodicUpdate: time.Unix(update, 0),
		IsCompleted:        comp == 1,
	}
}
