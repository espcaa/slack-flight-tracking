package shared

import (
	"flight-tracker-slack/flights"
	"fmt"
	"math"
	"strings"
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
		FlightID:         id,
		Status:           details.FlightStatus,
		OriginGate:       details.Origin.Gate,
		DestGate:         details.Destination.Gate,
		DepScheduled:     safeUnix(schedule.DepartureScheduled),
		DepEstimated:     safeUnix(schedule.DepartureEstimated),
		DepActual:        safeUnix(schedule.DepartureActual),
		TakeOffActual:    safeUnix(schedule.TakeOffActual),
		TakeOffEstimated: safeUnix(schedule.TakeOffEstimated),
		LandingActual:    safeUnix(schedule.LandingActual),
		ArrScheduled:     safeUnix(schedule.ArrivalScheduled),
		ArrEstimated:     safeUnix(schedule.ArrivalEstimated),
		ArrActual:        safeUnix(schedule.ArrivalActual),
	}
}

func FormatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	return strings.Join(parts, " ")
}

func GenerateProgressBar(width int, percentage float64) string {
	percentage = math.Max(0, math.Min(100, percentage))

	fullBlocksCount := int(math.Round((float64(width) * percentage) / 100))
	emptyBlocksCount := width - fullBlocksCount

	fullBlock := "█"
	emptyBlock := "░"

	bar := strings.Repeat(fullBlock, fullBlocksCount) + strings.Repeat(emptyBlock, emptyBlocksCount)

	return fmt.Sprintf("`%s` %.1f%%", bar, percentage)
}
