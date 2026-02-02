package commands

import (
	"flight-tracker-slack/shared"

	"github.com/slack-go/slack"
)

var ListCommand = shared.Command{
	Name:        "list-flights",
	Description: "List all tracked flights",
	Usage:       "/list-flights",
	Execute:     List,
}

func List(slashCommand slack.SlashCommand, config shared.Config) ([]slack.Block, bool, func()) {
	// list logic here
	return []slack.Block{}, true, nil
}
