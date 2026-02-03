package commands

import (
	"flight-tracker-slack/flights"
	"flight-tracker-slack/maps"
	"flight-tracker-slack/shared"
	"fmt"
	"os"
	"time"

	"github.com/google/shlex"
	"github.com/slack-go/slack"
)

var InfoCommand = shared.Command{
	Name:        "flight-info",
	Description: "Get information about a specific flight",
	Usage:       "/flight-info [flight_number]",
	Execute:     FlightInfo,
}

func FlightInfo(slashCommand slack.SlashCommand, config shared.Config) ([]slack.Block, bool, func() error) {

	// this is sent with the uploaded image
	blocks := []slack.Block{}
	// responseBlocks are sent immediatly as ephemeral message
	responseBlocks := []slack.Block{}

	args, err := shlex.Split(slashCommand.Text)
	if err != nil || len(args) < 1 {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Usage: `/flight-info [flight_number]`", false, false),
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

	flightNumber, err = flights.ExpandFlightNumber(flightNumber)
	if err != nil {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Could not expand flight number :x:", false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("_Error details:_ ```%v```", err), false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	flightInfo, err := flights.GetFlightInfo(flightNumber)
	if err != nil {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "We were unable to retrieve information for that flight :x:", false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("_Error details:_ ```%v```", err), false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	var fd flights.FlightDetail
	for _, f := range flightInfo.Flights {
		fd = f
		break
	}

	if fd.Origin.Iata == "" {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "No active flight found for that flight number :pensive:", false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	picturePath, err := maps.GenerateMapFromFlightDetail(config.TileStore, fd)

	if err != nil {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "We were unable to generate the flight map :x:", false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("_Error details:_ ```%v```", err), false, false),
				nil,
				nil,
			),
		}, false, nil
	}

	after := func() error {

		file, err := os.Open(picturePath)
		if err != nil {
			return err
		}
		defer file.Close()
		defer os.Remove(picturePath)

		byteSize, err := file.Stat()
		if err != nil {
			return err
		}

		fileSize := byteSize.Size()

		uploadResponse, err := config.SlackClient.UploadFileV2(slack.UploadFileV2Parameters{
			Channel:  slashCommand.ChannelID,
			File:     picturePath,
			Filename: picturePath,
			Reader:   file,
			FileSize: int(fileSize),
			Title:    fmt.Sprintf("%s - %s", flightNumber, time.Now().Format("2006-01-02")),
			Blocks: slack.Blocks{
				BlockSet: blocks,
			},
		})
		if err != nil {
			return err
		}
		fmt.Printf("Uploaded file: %+v\n", uploadResponse)
		return nil
	}

	airlineName := fd.Airline.FullName

	origin := fd.Origin.FriendlyLocation
	destination := fd.Destination.FriendlyLocation

	altitude := fd.Altitude
	speed := fd.Groundspeed

	schedule := fd.GetSchedule()
	departureScheduled := schedule.DepartureScheduled.Format("15:04") // HH:MM
	arrivalScheduled := schedule.ArrivalScheduled.Format("15:04")
	actualDeparture := schedule.DepartureActual.Format("15:04")
	estimatedArrival := schedule.ArrivalEstimated.Format("15:04")

	headerText := "*" + airlineName + " Flight " + flightNumber + "*"
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
		nil,
		nil,
	))

	infoText := "*From:* " + origin + "\n" +
		"*To:* " + destination + "\n" +
		"*Departure:* " + actualDeparture + " (scheduled: " + departureScheduled + ")\n" +
		"*Arrival:* " + arrivalScheduled + " (estimated: " + estimatedArrival + ")\n" +
		"*Altitude:* " + fmt.Sprintf("%d ft", altitude) + "\n" +
		"*Speed:* " + fmt.Sprintf("%d knots", speed)

	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, infoText, false, false),
		nil,
		nil,
	))

	responseBlocks = append(responseBlocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, "processing...", false, false),
		nil,
		nil,
	))

	return responseBlocks, false, after
}
