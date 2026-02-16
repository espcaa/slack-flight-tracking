package commands

import (
	"flight-tracker-slack/shared"
	"time"

	"github.com/slack-go/slack"
)

var ListCommand = shared.Command{
	Name:        "list-flights",
	Description: "List all tracked flights",
	Usage:       "/list-flights",
	Execute:     List,
}

func List(slashCommand slack.SlashCommand, config shared.Config) ([]slack.Block, bool, func() error) {
	var filter = shared.FlightFilter{
		SlackUserID: slashCommand.UserID,
	}
	var flightData, err = shared.GetFlights(filter, config)
	if err != nil {
		return shared.NewErrorBlocks(err), false, nil
	}
	if len(flightData) == 0 {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "You don't have any tracked flights yet. Use `/track-flight` to start tracking!", false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	var blocks []slack.Block
	for _, flight := range flightData {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, "• *"+flight.FlightNumber+"* in channel <#"+flight.SlackChannel+">, departing in "+shared.FormatDuration(time.Until(time.Unix(flight.Departure, 0))), false, false),
			nil,
			slack.NewAccessory(
				slack.NewButtonBlockElement(
					"untrack-"+flight.FlightNumber+"-"+flight.SlackChannel,
					"",
					slack.NewTextBlockObject(slack.PlainTextType, "Untrack", false, false),
				).WithStyle(slack.StyleDanger),
			),
		))
	}

	return blocks, false, nil
}
