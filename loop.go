package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"flight-tracker-slack/flights"
	"flight-tracker-slack/maps"
	"flight-tracker-slack/shared"
	"fmt"
	"image"
	"image/png"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/slack-go/slack"
)

type LogicLoop struct {
	Config        shared.Config
	flightCancels map[string]context.CancelFunc
	mu            sync.Mutex
}

func NewLogicLoop(cfg shared.Config) *LogicLoop {
	return &LogicLoop{
		Config:        cfg,
		flightCancels: make(map[string]context.CancelFunc),
	}
}

func (b *LogicLoop) Run() {
	log.Println("Starting logic loop...")

	flights, err := shared.GetFlights(shared.FlightFilter{
		DepartureAfter: time.Now().UTC().Unix(),
	}, b.Config)
	if err != nil {
		log.Println("Error loading flights from database:", err)
		return
	}

	log.Printf("Loaded %d flights from database\n", len(flights))

	for _, f := range flights {
		log.Printf("Tracking flight %s with departure at %s\n", f.ID, time.Unix(f.Departure, 0).Format(time.Kitchen))
		b.addFlight(f)
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		b.syncFlights()
	}
}

func WasAlertSent(flightID, alertType string, config shared.Config) bool {
	row := config.UserDB.QueryRow("SELECT 1 FROM alerts_sent WHERE flight_id = ? AND alert_type = ?", flightID, alertType)
	var exists int
	return row.Scan(&exists) == nil
}

func (b *LogicLoop) syncFlights() {
	flights, err := shared.GetFlights(shared.FlightFilter{}, b.Config)
	if err != nil {
		log.Println("Error loading flights from database:", err)
		return
	}

	dbIDs := make(map[string]bool, len(flights))
	for _, f := range flights {
		// if flight departure is in less than 4 hours and it's not being tracked, start tracking it
		if time.Until(time.Unix(f.Departure, 0)) < 4*time.Hour {
			if _, exists := b.flightCancels[f.ID]; !exists {
				log.Printf("New flight to track: %s departing at %s\n", f.ID, time.Unix(f.Departure, 0).Format(time.Kitchen))
				b.addFlight(f)
			}
		} else {
			log.Printf("Flight %s departure is more than 4 hours away, not tracking yet\n", f.ID)
		}
	}

	b.mu.Lock()
	for id, cancel := range b.flightCancels {
		if !dbIDs[id] {
			log.Printf("Flight %s is no longer active, removing from tracking\n", id)
			cancel()
			delete(b.flightCancels, id)
		}
	}
	b.mu.Unlock()
}

func (b *LogicLoop) trackFlight(ctx context.Context, f shared.Flight) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping flight:", f.ID)
			return
		case <-ticker.C:
			log.Printf("tick for flight %s\n", f.ID)
			data, err := flights.GetFlightInfo(f.FlightNumber)
			if err != nil || data.GetFirstFlight() == nil {
				continue
			}
			currData := data.GetFirstFlight()

			curr := shared.FlightDetailsToFlightState(currData, f.ID)
			prev, err := shared.GetFlightState(f.ID, b.Config)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				log.Println("Error getting flight state for", f.ID, ":", err)
				continue
			}

			b.detectChanges(f, prev, &curr, currData)

			shared.SaveFlightState(curr, b.Config)

		}
	}
}

func (b *LogicLoop) detectChanges(f shared.Flight, prev *shared.FlightState, curr *shared.FlightState, currData *flights.FlightDetail) {

	destinationTimezone := strings.TrimPrefix(currData.Destination.TZ, ":")
	destLoc, err := time.LoadLocation(destinationTimezone)
	if err != nil {
		log.Printf("Error loading destination timezone %q: %v\n", currData.Destination.TZ, err)
		destLoc = time.UTC
	}
	departureTimezone := strings.TrimPrefix(currData.Origin.TZ, ":")
	depLoc, err := time.LoadLocation(departureTimezone)
	if err != nil {
		log.Printf("Error loading departure timezone %q: %v\n", currData.Origin.TZ, err)
		depLoc = time.UTC
	}

	if prev == nil {
		log.Printf("No previous state for flight %s, skipping change detection\n", f.ID)
		return
	}

	// check if dep gate was announced
	if curr.OriginGate != "" && WasAlertSent(f.ID, "departure_gate_announced", b.Config) == false {
		depTime := time.Unix(curr.DepEstimated, 0).In(depLoc).Format(time.Kitchen)
		depIn := shared.FormatDuration(time.Until(time.Unix(curr.DepEstimated, 0)))
		b.sendAlert(f, "departure_gate_announced", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*:seat: Gate announced!* :seat:\nGate *%s*\nEstimated departure time: %s (_in %s_)", curr.OriginGate, depTime, depIn), false, false),
			nil,
			nil,
		), nil)
	}

	// check if the flight departed from gate

	if curr.DepActual != 0 && WasAlertSent(f.ID, "flight_departed_from_gate", b.Config) == false {
		depTime := time.Unix(curr.DepActual, 0).In(depLoc).Format(time.Kitchen)
		depEstimated := time.Unix(curr.DepEstimated, 0).In(depLoc).Format(time.Kitchen)
		gateMsg := ""
		if curr.OriginGate != "" {
			gateMsg = fmt.Sprintf("gate %s", curr.OriginGate)
		} else {
			gateMsg = "the gate"
		}

		taxiMsg := ""
		estimatedTaxiTime := curr.TakeOffEstimated - curr.DepEstimated
		if estimatedTaxiTime > 0 {
			taxiMsg = fmt.Sprintf("Estimated taxi time: %s", shared.FormatDuration(time.Duration(estimatedTaxiTime)*time.Second))
		}
		b.sendAlert(f, "flight_departed_from_gate", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*:airplane: Flight departed from %s! :airplane:\nDeparture time: ~%s~ %s* \n %s", gateMsg, depEstimated, depTime, taxiMsg), false, false),
			nil,
			nil,
		), nil)
	}

	// check if the flight took off

	if curr.TakeOffActual != 0 && WasAlertSent(f.ID, "flight_takeoff", b.Config) == false {
		takeOffTime := time.Unix(curr.TakeOffActual, 0).In(depLoc).Format(time.Kitchen)
		takeOffEstimated := time.Unix(curr.TakeOffEstimated, 0).In(depLoc).Format(time.Kitchen)
		flightEstimatedDuration := curr.ArrEstimated - curr.DepEstimated
		b.sendAlert(f, "flight_takeoff", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":airplane_departure: *Flight took off!* :airplane_departure:\nTakeoff time: ~%s~ %s \n Estimated flight duration: %s", takeOffEstimated, takeOffTime, shared.FormatDuration(time.Duration(flightEstimatedDuration)*time.Second)), false, false),
			nil,
			nil,
		), nil)
	}

	// check if flight landed

	if curr.LandingActual != 0 && WasAlertSent(f.ID, "flight_landed", b.Config) == false {
		// if gate is available, include it in the message
		var gateMsg string
		if curr.DestGate != "" {
			gateMsg = fmt.Sprintf("\n taxiing to gate %s", curr.DestGate)
		} else {
			gateMsg = ""
		}
		arrTime := time.Unix(curr.ArrActual, 0).In(destLoc).Format(time.Kitchen)
		arrEstimated := time.Unix(curr.ArrEstimated, 0).In(destLoc).Format(time.Kitchen)
		b.sendAlert(f, "flight_landed", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":airplane_arriving: *Flight landed!* :airplane_arriving:\nLanding time: ~%s~ %s%s", arrEstimated, arrTime, gateMsg), false, false),
			nil,
			nil,
		), nil)
		log.Printf("Flight %s has landed, stopping tracking\n", f.ID)
		b.removeFlight(f.ID)
		return
	}

	// check if flight arrived at gate
	// if it did, remove it from tracking & db
	if curr.ArrActual != 0 && WasAlertSent(f.ID, "flight_arrived_at_gate", b.Config) == false {
		var gateMsg string
		if curr.DestGate != "" {
			gateMsg = fmt.Sprintf(" at gate %s", curr.DestGate)
		} else {
			gateMsg = ""
		}
		arrTime := time.Unix(curr.ArrActual, 0).In(destLoc).Format(time.Kitchen)
		arrEstimated := time.Unix(curr.ArrEstimated, 0).In(destLoc).Format(time.Kitchen)
		b.sendAlert(f, "flight_arrived_at_gate", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":airplane: *Flight arrived%s* :airplane:\nArrival time: ~%s~ %s", gateMsg, arrEstimated, arrTime), false, false),
			nil,
			nil,
		), nil)
		log.Printf("Flight %s has arrived at gate, stopping tracking\n", f.ID)
		b.removeFlight(f.ID)
		return
	}

	// regular updates during the flight (1 every 2 hours)
	if curr.DepActual != 0 && curr.ArrActual == 0 {
		hoursSinceDeparture := int(time.Since(time.Unix(curr.DepActual, 0)).Hours())
		window := hoursSinceDeparture / 2

		if window > 0 {
			alertID := fmt.Sprintf("in_flight_update_%d", window)
			image, err := maps.GenerateMapFromFlightDetail(b.Config.TileStore, *currData)
			if err != nil {
				log.Printf("Error generating map for flight %s: %v", f.ID, err)
			}
			progressBar := shared.GenerateProgressBar(10, float64(time.Since(time.Unix(curr.DepActual, 0)).Seconds())/float64(curr.ArrEstimated-curr.DepActual))
			timeLeft := shared.FormatDuration(time.Until(time.Unix(curr.ArrEstimated, 0)))
			b.sendAlert(f, alertID, slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, ":airplane: *Still flying!* :airplane:\n "+progressBar+"\n("+timeLeft+" left)", false, false),
				nil,
				nil,
			), image)
		}
	}

	// check if departure_time is updated
	if prev.DepEstimated != curr.DepEstimated {
		prevTime := time.Unix(prev.DepEstimated, 0).In(depLoc).Format(time.Kitchen)
		currTime := time.Unix(curr.DepEstimated, 0).In(depLoc).Format(time.Kitchen)
		b.sendAlert(f, fmt.Sprintf("departure_time_change_%d", curr.DepEstimated), slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":rotating_light: *Departure time updated!* :rotating_light:\nPrevious: %s\nNew: %s", prevTime, currTime), false, false),
			nil,
			nil,
		), nil)
	}
	// check if gate was updated
	if prev.OriginGate != curr.OriginGate && curr.OriginGate != "" {
		b.sendAlert(f, fmt.Sprintf("gate_change_%s", curr.OriginGate), slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":rotating_light: *Gate updated!* :rotating_light:\nPrevious: %s\nNew: %s", prev.OriginGate, curr.OriginGate), false, false),
			nil,
			nil,
		), nil)
	}
	// check if arrival gate was updated
	if prev.DestGate != curr.DestGate {
		b.sendAlert(f, fmt.Sprintf("arrival_gate_change_%s", curr.DestGate), slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":rotating_light: *Arrival gate updated!* :rotating_light:\nPrevious: %s\nNew: %s", prev.DestGate, curr.DestGate), false, false),
			nil,
			nil,
		), nil)
	}
	// check if arrival time is updated
	if prev.ArrEstimated != curr.ArrEstimated {
		prevTime := time.Unix(prev.ArrEstimated, 0).In(destLoc).Format(time.Kitchen)
		currTime := time.Unix(curr.ArrEstimated, 0).In(destLoc).Format(time.Kitchen)
		b.sendAlert(f, fmt.Sprintf("arrival_time_change_%d", curr.ArrEstimated), slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":rotating_light: *Arrival time updated!* :rotating_light:\nPrevious: %s\nNew: %s", prevTime, currTime), false, false),
			nil,
			nil,
		), nil)
	}

}

func (b *LogicLoop) sendAlert(f shared.Flight, alertType string, blocks slack.Block, image *image.RGBA) {
	// add footer to blocks

	footer := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("_flight %s - %s, tracked by <@%s>_", f.FlightNumber, f.ID, f.SlackUserID), false, false),
	)

	if shared.AlertAlreadySent(f.ID, alertType, b.Config) {
		return
	}

	if image != nil {
		var buf bytes.Buffer
		if err := png.Encode(&buf, image); err != nil {
			log.Printf("Error encoding image for flight %s (%s): %v", f.ID, alertType, err)
			return
		}
		uploadResponse, err := b.Config.SlackClient.UploadFileV2(slack.UploadFileV2Parameters{
			Channel:  f.SlackChannel,
			Filename: "flight_map.png",
			Reader:   &buf,
			FileSize: buf.Len(),
			Title:    fmt.Sprintf("%s - %s", f.FlightNumber, time.Now().Format("2006-01-02")),
			Blocks: slack.Blocks{
				BlockSet: []slack.Block{blocks, footer},
			},
		})
		if err != nil {
			log.Printf("Error uploading image for flight %s (%s): %v", f.ID, alertType, err)
		}
		fmt.Printf("Uploaded file: %+v\n", uploadResponse)

	} else {
		_, _, err := b.Config.SlackClient.PostMessage(
			f.SlackChannel,
			slack.MsgOptionBlocks(blocks, footer),
		)
		if err != nil {
			log.Printf("Error sending alert for flight %s (%s): %v", f.ID, alertType, err)
			return
		}
	}

	shared.MarkAlertSent(f.ID, alertType, b.Config)
	log.Printf("Alert sent for flight %s: %s", f.ID, alertType)
}

func (b *LogicLoop) addFlight(f shared.Flight) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.flightCancels[f.ID]; exists {
		log.Println("Flight already tracked:", f.ID)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.flightCancels[f.ID] = cancel

	go b.trackFlight(ctx, f)
	log.Println("Started tracking flight:", f.ID)
}

func (b *LogicLoop) removeFlight(flightID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if cancel, exists := b.flightCancels[flightID]; exists {
		cancel()
		delete(b.flightCancels, flightID)
		log.Println("Stopped tracking flight:", flightID)
	}

	// delete from the database as well
	err := shared.UntrackFlight(flightID, b.Config)
	if err != nil {
		log.Printf("Error untracking flight %s from database: %v", flightID, err)
	}
}
