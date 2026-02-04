package commands

import (
	"flight-tracker-slack/shared"

	"github.com/slack-go/slack"
)

var UntrackCommand = shared.Command{
	Name:        "untrack-flight",
	Description: "Untrack a flight",
	Usage:       "/untrack-flight [flight_number] [channel (optional)]",
	Execute:     Untrack,
}

func Untrack(slashCommand slack.SlashCommand, config shared.Config) ([]slack.Block, bool, func(responseURL string) error) {
	return []slack.Block{}, true, nil
}
