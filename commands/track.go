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

func Track(slashCommand slack.SlashCommand, config shared.Config) ([]slack.Block, bool, func(responseURL string) error) {
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
		return NewErrorBlocks(err), false, nil
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
					"track_departure_date_picker",
				),
			),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, "_Departure time_:", false, false),
			nil,
			slack.NewAccessory(
				slack.NewTimePickerBlockElement(
					"track_departure_time_picker",
				),
			),
		),
		slack.NewActionBlock(
			"track_flight_submit_button",
			slack.NewButtonBlockElement(
				"track_flight_submit",
				flightNumber,
				slack.NewTextBlockObject(slack.PlainTextType, "Track Flight", false, false),
			).WithStyle(slack.StylePrimary),
		),
	}

	return blocks, false, nil
}
