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

func List(commandText string, responseURL string, config shared.Config) ([]slack.Block, bool) {
	// list logic here
	return []slack.Block{}, true
}
