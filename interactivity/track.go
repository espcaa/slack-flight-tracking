package interactivity

import (
	"errors"
	"flight-tracker-slack/shared"
	"fmt"
	"log"
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

			// register the flight for tracking
			var departure_unix int64
			var departure_date_time time.Time
			departure_date_time, err := time.Parse("2006-01-02", selectedDate)
			if err != nil {
				log.Printf("Error parsing departure date: %v\n", err)
				return
			}
			departure_time, err := time.Parse("15:04", selectedTime)
			if err != nil {
				log.Printf("Error parsing departure time: %v\n", err)
				return
			}
			departure_date_time = departure_date_time.Add(time.Hour*time.Duration(departure_time.Hour()) + time.Minute*time.Duration(departure_time.Minute()))
			departure_unix = departure_date_time.Unix()

			var flight shared.Flight = shared.Flight{
				ID:           uuid.New().String(),
				FlightNumber: flightNum,
				Departure:    time.Unix(departure_unix, 0).Format(time.RFC3339),
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
