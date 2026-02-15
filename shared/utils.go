package shared

import (
	"flight-tracker-slack/flights"
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

func NewErrorBlocks(err error, customMessage ...string) []slack.Block {
	message := "Something wrong happened :x:"
	if len(customMessage) > 0 && customMessage[0] != "" {
		message = customMessage[0]
	}

	return []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, message, false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("_Error details:_ ```%v```", err), false, false),
			nil,
			nil,
		),
	}
}

func safeUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func FlightDetailsToFlightState(details *flights.FlightDetail, id string) FlightState {

	schedule := details.GetSchedule()

	return FlightState{
		FlightID:     id,
		Status:       details.FlightStatus,
		OriginGate:   details.Origin.Gate,
		DestGate:     details.Destination.Gate,
		DepScheduled: safeUnix(schedule.DepartureScheduled),
		DepEstimated: safeUnix(schedule.DepartureEstimated),
		DepActual:    safeUnix(schedule.DepartureActual),
		ArrScheduled: safeUnix(schedule.ArrivalScheduled),
		ArrEstimated: safeUnix(schedule.ArrivalEstimated),
		ArrActual:    safeUnix(schedule.ArrivalActual),
	}
}
