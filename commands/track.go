package commands

import (
	"flight-tracker-slack/shared"

	"github.com/google/shlex"
	"github.com/slack-go/slack"
)

var TrackCommand = shared.Command{
	Name:        "track-flight",
	Description: "Track a flight",
	Usage:       "/track-flight [flight_number (iata or icao)] [date (optional, any format)] [channel (optional)]",
	Execute:     Track,
}

func Track(slashCommand slack.SlashCommand, config shared.Config) ([]slack.Block, bool, func()) {
	// parse the arguments from commandText
	// separate by spaces or "" when multiple words

	args, err := shlex.Split(slashCommand.Text)
	if err != nil || len(args) < 1 {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Usage: `/track [flight_number (iata or icao)] [date (optional, any format)] [channel (optional)]`", false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	flightNumber := args[0]
	var date string
	var channel string

	_ = flightNumber // to avoid unused variable error
	_ = date
	_ = channel

	if len(args) >= 2 {
		date = args[1]
	}
	if len(args) >= 3 {
		channel = args[2]
	}

	// if there's a date, parse it as a
	return []slack.Block{}, true, nil
}
