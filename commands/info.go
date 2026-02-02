package commands

import (
	"flight-tracker-slack/flights"
	"flight-tracker-slack/shared"
	"fmt"

	"github.com/google/shlex"
	"github.com/slack-go/slack"
)

var InfoCommand = shared.Command{
	Name:        "flight-info",
	Description: "Get information about a specific flight",
	Usage:       "/flight-info [flight_number]",
	Execute:     FlightInfo,
}

func FlightInfo(commandText string, responseURL string, config shared.Config) []slack.Block {
	// get the arguments

	args, err := shlex.Split(commandText)
	if err != nil || len(args) < 1 {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Usage: `/flight-info [flight_number]`", false, false),
				nil,
				nil,
			),
		}
	}

	flightNumber := args[0]

	// number wizzardry here

	// check if it's an iata or icao code

	if !flights.IcaoPattern.MatchString(flightNumber) && !flights.IataPattern.MatchString(flightNumber) {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Invalid flight number format. Please provide a valid IATA or ICAO flight number.", false, false),
				nil,
				nil,
			),
		}
	} else if flights.IataPattern.MatchString(flightNumber) {
		// convert to icao
		icao, err := flights.ExpandFlightNumber(flightNumber)
		if err != nil {
			return []slack.Block{
				slack.NewSectionBlock(
					slack.NewTextBlockObject(slack.MarkdownType, "Could not convert IATA to ICAO flight number.", false, false),
					nil,
					nil,
				),
			}
		}
		flightNumber = icao
	}

	// fetch flight info
	flightInfo, err := flights.GetFlightInfo(flightNumber)
	if err != nil {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Could not retrieve flight information. Please ensure the flight number is correct.", false, false),
				nil,
				nil,
			),
		}
	}

	var fd flights.FlightDetail
	for _, f := range flightInfo.Flights {
		fd = f
		break
	}

	// build response blocks

	airlineName := fd.Airline.FullName

	origin := fd.Origin.FriendlyLocation
	destination := fd.Destination.FriendlyLocation

	altitude := fd.Altitude
	speed := fd.Groundspeed

	schedule := fd.GetSchedule()
	departureScheduled := schedule.DepartureScheduled.Format("15:04") // HH:MM
	arrivalScheduled := schedule.ArrivalScheduled.Format("15:04")

	blocks := []slack.Block{}

	headerText := "*" + airlineName + " Flight " + flightNumber + "*"
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
		nil,
		nil,
	))

	infoText := "*From:* " + origin + "\n" +
		"*To:* " + destination + "\n" +
		"*Scheduled Departure:* " + departureScheduled + "\n" +
		"*Scheduled Arrival:* " + arrivalScheduled + "\n" +
		"*Altitude:* " + fmt.Sprintf("%d ft", altitude) + "\n" +
		"*Speed:* " + fmt.Sprintf("%d knots", speed)

	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, infoText, false, false),
		nil,
		nil,
	))

	return blocks
}
