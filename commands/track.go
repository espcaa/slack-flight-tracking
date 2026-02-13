package commands

import (
	"flight-tracker-slack/flights"
	"flight-tracker-slack/shared"

	"github.com/google/shlex"
	"github.com/slack-go/slack"
)

var TrackCommand = shared.Command{
	Name:        "track-flight",
	Description: "Track a flight",
	Usage:       "/track-flight [flight_number (iata or icao)]",
	Execute:     Track,
}

func Track(slashCommand slack.SlashCommand, config shared.Config) ([]slack.Block, bool, func() error) {

	// first check if the bot is in the channel, and it's not a dm

	isInChannel, err := config.SlackClient.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID:         slashCommand.ChannelID,
		IncludeLocale:     false,
		IncludeNumMembers: false,
	})
	if err != nil || isInChannel.IsMember == false && isInChannel.IsOpen == false {
		return shared.NewErrorBlocks(err, ":warning: I need to be in this channel to track flights here! "), false, nil
	}

	args, err := shlex.Split(slashCommand.Text)
	if err != nil || len(args) < 1 {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "This didn't work :pensive: \n _Usage: `/track [flight_number (iata or icao)]`_", false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	flightNumber := args[0]

	if !flights.FlightNumPattern.MatchString(flightNumber) {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Doesn't look like a valid flight number... :pensive:", false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "_Flight numbers usually look like `AA100` or `DLH400`._", false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	flightsInfo, err := flights.GetFlightInfo(flightNumber)
	if err != nil {
		return shared.NewErrorBlocks(err), false, nil
	}
	if flightsInfo.GetFirstFlight().Airline.FullName == "" {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Hmm... I couldn't find any flight with that number :pensive:", false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "_Please double-check the flight number and try again._", false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	// now return a datepicker

	flight := flightsInfo.GetFirstFlight()
	airlineName := flight.Airline.FullName
	departure := flight.Origin.FriendlyLocation
	arrival := flight.Destination.FriendlyLocation

	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, ":beverage_box: So you're taking a flight from *"+departure+"* to *"+arrival+"* with *"+airlineName+"*?", false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, "Sounds great! We just need a bit more info...", false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, "_Departure date_:", false, false),
			nil,
			slack.NewAccessory(
				slack.NewDatePickerBlockElement(
					"trackflightformsubmit-departuredate",
				),
			),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, "_Departure time_: (choose the nearest time to your actual departure time, but before it)", false, false),
			nil,
			slack.NewAccessory(
				slack.NewTimePickerBlockElement(
					"trackflightformsubmit-departuretime",
				),
			),
		),
		slack.NewActionBlock(
			"track_flight_submit_button",
			slack.NewButtonBlockElement(
				"trackflightformsubmit-button",
				flightNumber,
				slack.NewTextBlockObject(slack.PlainTextType, "Track Flight", false, false),
			).WithStyle(slack.StylePrimary),
		),
	}

	return blocks, false, nil
}
