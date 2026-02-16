package main

import (
	"context"
	"database/sql"
	"errors"
	"flight-tracker-slack/flights"
	"flight-tracker-slack/shared"
	"fmt"
	"log"
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
		dbIDs[f.ID] = true
		b.addFlight(f)
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
	ticker := time.NewTicker(5 * time.Second)
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

			b.detectChanges(f, prev, &curr)

			shared.SaveFlightState(curr, b.Config)

		}
	}
}

func (b *LogicLoop) detectChanges(f shared.Flight, prev *shared.FlightState, curr *shared.FlightState) {
	if prev == nil {
		log.Printf("No previous state for flight %s, skipping change detection\n", f.ID)
		return
	}

	// check if dep gate was just announced
	if curr.DepGate != "" && WasAlertSent(f.ID, "departure_gate_announced", b.Config) == false {
		depTime := time.Unix(curr.DepEstimated, 0).Format(time.Kitchen)
		b.sendAlert(f, "departure_gate_announced", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*:seat: Gate announced!* :seat:\nEstimated departure time: %s\nGate: %s", depTime, curr.DepGate), false, false),
			nil,
			nil,
		))
	}

	// check if the flight departed from gate

	if curr.DepActual != 0 && WasAlertSent(f.ID, "flight_departed_from_gate", b.Config) == false {
		depTime := time.Unix(curr.DepActual, 0).Format(time.Kitchen)
		depEstimated := time.Unix(curr.DepEstimated, 0).Format(time.Kitchen)
		gateMsg := ""
		if curr.OriginGate != "" {
			gateMsg = fmt.Sprintf("gate %s", curr.OriginGate)
		} else {
			gateMsg = "the gate"
		}

		taxiMsg := ""
		estimatedTaxiTime := curr.TakeOffEstimated - curr.DepEstimated
		if estimatedTaxiTime > 0 {
			taxiMsg = fmt.Sprintf(" (estimated taxi time: %d mins)", estimatedTaxiTime/60)
		}
		b.sendAlert(f, "flight_departed_from_gate", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*:airplane: Flight departed from %s!%s :airplane_departure:\nDeparture time: ~%s~ %s*", gateMsg, taxiMsg, depEstimated, depTime), false, false),
			nil,
			nil,
		))
	}

	// check if the flight took off

	if curr.TakeOffActual != 0 && WasAlertSent(f.ID, "flight_takeoff", b.Config) == false {
		takeOffTime := time.Unix(curr.TakeOffActual, 0).Format(time.Kitchen)
		takeOffEstimated := time.Unix(curr.TakeOffEstimated, 0).Format(time.Kitchen)
		flightEstimatedDuration := curr.ArrEstimated - curr.DepEstimated
		b.sendAlert(f, "flight_takeoff", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":airplane_departure: *Flight took off!* :airplane_departure:\nTakeoff time: ~%s~ %s \n Estimated flight duration: %s", takeOffEstimated, takeOffTime, shared.FormatDuration(time.Duration(flightEstimatedDuration)*time.Second)), false, false),
			nil,
			nil,
		))
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
		arrTime := time.Unix(curr.ArrActual, 0).Format(time.Kitchen)
		arrEstimated := time.Unix(curr.ArrEstimated, 0).Format(time.Kitchen)
		b.sendAlert(f, "flight_landed", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":airplane_arriving: *Flight landed!* :airplane_arriving:\nLanding time: ~%s~ %s%s", arrEstimated, arrTime, gateMsg), false, false),
			nil,
			nil,
		))
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
		arrTime := time.Unix(curr.ArrActual, 0).Format(time.Kitchen)
		arrEstimated := time.Unix(curr.ArrEstimated, 0).Format(time.Kitchen)
		b.sendAlert(f, "flight_arrived_at_gate", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":airplane: *Flight arrived%s* :airplane:\nArrival time: ~%s~ %s", gateMsg, arrTime, arrEstimated), false, false),
			nil,
			nil,
		))
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

			b.sendAlert(f, alertID, slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, ":airplane: *In-flight update!* Still cruising... :airplane:", false, false),
				nil,
				nil,
			))
		}
	}

	// check if departure_time is updated
	if prev.DepEstimated != curr.DepEstimated {
		prevTime := time.Unix(prev.DepEstimated, 0).Format(time.Kitchen)
		currTime := time.Unix(curr.DepEstimated, 0).Format(time.Kitchen)
		b.sendAlert(f, fmt.Sprintf("departure_time_change_%d", curr.DepEstimated), slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":rotating_light: *Departure time updated!* :rotating_light:\nPrevious: %s\nNew: %s", prevTime, currTime), false, false),
			nil,
			nil,
		))
	}
	// check if gate was updated
	if prev.OriginGate != curr.OriginGate {
		b.sendAlert(f, fmt.Sprintf("gate_change_%s", curr.OriginGate), slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":rotating_light: *Gate updated!* :rotating_light:\nPrevious: %s\nNew: %s", prev.OriginGate, curr.OriginGate), false, false),
			nil,
			nil,
		))
	}
	// check if arrival gate was updated
	if prev.DestGate != curr.DestGate {
		b.sendAlert(f, fmt.Sprintf("arrival_gate_change_%s", curr.DestGate), slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":rotating_light: *Arrival gate updated!* :rotating_light:\nPrevious: %s\nNew: %s", prev.DestGate, curr.DestGate), false, false),
			nil,
			nil,
		))
	}
	// check if arrival time is updated
	if prev.ArrEstimated != curr.ArrEstimated {
		prevTime := time.Unix(prev.ArrEstimated, 0).Format(time.Kitchen)
		currTime := time.Unix(curr.ArrEstimated, 0).Format(time.Kitchen)
		b.sendAlert(f, fmt.Sprintf("arrival_time_change_%d", curr.ArrEstimated), slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":rotating_light: *Arrival time updated!* :rotating_light:\nPrevious: %s\nNew: %s", prevTime, currTime), false, false),
			nil,
			nil,
		))
	}

}

func (b *LogicLoop) sendAlert(f shared.Flight, alertType string, blocks slack.Block) {
	// add footer to blocks

	footer := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("_flight %s - %s, tracked by <@%s>_", f.FlightNumber, f.ID, f.SlackUserID), false, false),
	)

	if shared.AlertAlreadySent(f.ID, alertType, b.Config) {
		return
	}

	_, _, err := b.Config.SlackClient.PostMessage(
		f.SlackChannel,
		slack.MsgOptionBlocks(blocks, footer),
	)
	if err != nil {
		log.Printf("Error sending alert for flight %s (%s): %v", f.ID, alertType, err)
		return
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
