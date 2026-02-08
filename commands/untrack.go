package commands

import (
	"flight-tracker-slack/shared"

	"github.com/google/shlex"
	"github.com/slack-go/slack"
)

var UntrackCommand = shared.Command{
	Name:        "untrack-flight",
	Description: "Untrack a flight",
	Usage:       "/untrack-flight [flight_number] [channel (optional)]",
	Execute:     Untrack,
}

func Untrack(slashCommand slack.SlashCommand, config shared.Config) ([]slack.Block, bool, func() error) {
	args, err := shlex.Split(slashCommand.Text)
	if err != nil || len(args) < 1 {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "This didn't work :pensive: \n _Usage: `/untrack-flight [flight_number] [channel (optional)]`_ \n You can also use `/list-flights` to see all your tracked flights.", false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	flightNumber := args[0]
	var channelID string
	if len(args) >= 2 {
		channelID = args[1]
	} else {
		channelID = slashCommand.ChannelID
	}

	var filter = shared.FlightFilter{
		SlackUserID:  slashCommand.UserID,
		FlightNumber: flightNumber,
		SlackChannel: channelID,
	}
	flights, err := shared.GetFlights(filter, config)

	if err != nil {
		return shared.NewErrorBlocks(err), false, nil
	}

	if len(flights) == 0 {
		// no flight found to untrack, list all flights for the user corresponding to the flight number$
		var filter = shared.FlightFilter{
			SlackUserID:  slashCommand.UserID,
			FlightNumber: flightNumber,
		}
		flights, err := shared.GetFlights(filter, config)
		if err != nil {
			return shared.NewErrorBlocks(err), false, nil
		}
		if len(flights) == 0 {
			return []slack.Block{
				slack.NewSectionBlock(
					slack.NewTextBlockObject(slack.MarkdownType, "I couldn't find any tracked flight with that number :pensive:", false, false),
					nil,
					nil,
				),
			}, false, nil
		} else {
			var blocks []slack.Block
			blocks = append(blocks, slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "I couldn't find a tracked flight with that number in this channel, but here are the flights with that number that you're tracking in other channels:", false, false),
				nil,
				nil,
			))
			for _, flight := range flights {
				blocks = append(blocks, slack.NewSectionBlock(
					slack.NewTextBlockObject(slack.MarkdownType, "• *"+flight.FlightNumber+"* in channel <#"+flight.SlackChannel+">", false, false),
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
	}

	err = shared.UntrackFlight(flights[0].ID, config)
	if err != nil {
		return shared.NewErrorBlocks(err), false, nil
	}

	return []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, "Successfully untracked flight *"+flightNumber+"* in channel <#"+channelID+">.", false, false),
			nil,
			nil,
		),
	}, false, nil
}
