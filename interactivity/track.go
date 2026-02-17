package interactivity

import (
	"errors"
	"flight-tracker-slack/flights"
	"flight-tracker-slack/shared"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/slack-go/slack"
)

var TrackInteraction shared.Interaction = shared.Interaction{
	Prefix:  "trackflightformsubmit",
	Execute: HandleTrackFlightFormSubmit,
}

func HandleTrackFlightFormSubmit(payload slack.InteractionCallback, config shared.Config) {
	for _, action := range payload.ActionCallback.BlockActions {
		if action.ActionID == "trackflightformsubmit-button" {
			log.Printf("Received track flight form submit interaction with value: %s\n", action.Value)
			flightNum := action.Value
			_ = flightNum // use this to avoid unused variable error for now

			var selectedDate, selectedTime string
			if payload.BlockActionState == nil || payload.BlockActionState.Values == nil {
				log.Println("No state values found in the payload.")
				return
			}
			for _, block := range payload.BlockActionState.Values {
				if val, ok := block["trackflightformsubmit-departuredate"]; ok {
					selectedDate = val.SelectedDate
				}
				if val, ok := block["trackflightformsubmit-departuretime"]; ok {
					selectedTime = val.SelectedTime
				}
			}
			if selectedDate == "" || selectedTime == "" {
				config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(
					shared.NewErrorBlocks(errors.New("Please select both a departure date and time before submitting the form."))...,
				))
				return
			}

			// find the departure airport tz

			flightData, err := flights.GetFlightInfo(flightNum)
			if err != nil {
				log.Printf("Error fetching flight info for %s: %v\n", flightNum, err)
				config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(
					shared.NewErrorBlocks(fmt.Errorf("Could not fetch flight information for %s. Please check the flight number and try again.", flightNum))...,
				))
				return
			}
			firstFlight := flightData.GetFirstFlight()
			if firstFlight == nil {
				log.Printf("No flight data found for flight number %s\n", flightNum)
				config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(
					shared.NewErrorBlocks(fmt.Errorf("No flight information found for flight number %s. Please check the flight number and try again.", flightNum))...,
				))
				return
			}
			if firstFlight.Origin.Iata == "" {
				log.Printf("No active flight found for flight number %s\n", flightNum)
				config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(
					shared.NewErrorBlocks(fmt.Errorf("No active flight found for flight number %s. Please check the flight number and try again.", flightNum))...,
				))
				return
			}

			// register the flight for tracking
			var departureUnix int64
			var departureDateTime time.Time
			// remove the ":" at the start of the timezone string
			correctedTimezone := strings.TrimPrefix(firstFlight.Origin.TZ, ":")
			loc, err := time.LoadLocation(correctedTimezone)
			if err != nil {
				log.Printf("Error loading timezone %q: %v\n", firstFlight.Origin.TZ, err)
				loc = time.UTC
			}
			departureDateTime, err = time.ParseInLocation("2006-01-02", selectedDate, loc)
			if err != nil {
				log.Printf("Error parsing departure date: %v\n", err)
				return
			}
			departure_time, err := time.Parse("15:04", selectedTime)
			if err != nil {
				log.Printf("Error parsing departure time: %v\n", err)
				return
			}
			departureDateTime = time.Date(departureDateTime.Year(), departureDateTime.Month(), departureDateTime.Day(), departure_time.Hour(), departure_time.Minute(), 0, 0, loc)
			departureUnix = departureDateTime.Unix()

			var flight shared.Flight = shared.Flight{
				ID:           uuid.New().String(),
				FlightNumber: flightNum,
				Departure:    departureUnix,
				SlackChannel: payload.Channel.ID,
				SlackUserID:  payload.User.ID,
			}
			err = shared.RegisterTrackedFlight(flight, config)
			if err != nil {
				log.Printf("Error registering tracked flight: %v\n", err)
				config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(
					shared.NewErrorBlocks(err)...,
				))
			}

			// send a message to the channel confirming the tracking
			config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(
				slack.NewSectionBlock(
					slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Flight added for tracking in channel <#%s>! :airplane:", payload.Channel.ID), false, false),
					nil,
					nil,
				),
			))

			return
		}
	}
}
