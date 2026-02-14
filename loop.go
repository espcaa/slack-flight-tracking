package main

import (
	"context"
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

	flights, err := shared.GetFlights(shared.FlightFilter{}, b.Config)
	if err != nil {
		log.Println("Error loading flights from database:", err)
		return
	}

	log.Printf("Loaded %d flights from database\n", len(flights))

	for _, f := range flights {
		// add if flight departure is in the past + not already goroutine for it
		if time.Unix(f.Departure, 0).After(time.Now()) {
			log.Printf("Tracking flight %s with departure at %s\n", f.ID, time.Unix(f.Departure, 0).Format(time.Kitchen))
			b.addFlight(f)
		}
	}
	// remove flights not in database anymore every 30s
	tickerRemove := time.NewTicker(30 * time.Second)
	defer tickerRemove.Stop()

	go func() {
		for range tickerRemove.C {
			log.Println("Checking for flights to remove...")
			currentFlights, err := shared.GetFlights(shared.FlightFilter{}, b.Config)
			if err != nil {
				log.Println("Error loading flights from database:", err)
				continue
			}

			currentFlightIDs := make(map[string]bool)
			for _, f := range currentFlights {
				currentFlightIDs[f.ID] = true
			}

			b.mu.Lock()
			for trackedID := range b.flightCancels {
				if !currentFlightIDs[trackedID] {
					log.Printf("Flight %s is no longer in the database, removing from tracking\n", trackedID)
					b.removeFlight(trackedID)
				}
			}
			b.mu.Unlock()
		}
	}()

	// recheck every 30s to add new flights that are leaving now
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Checking for new flights to track...")
		flights, err := shared.GetFlights(shared.FlightFilter{}, b.Config)
		if err != nil {
			log.Println("Error loading flights from database:", err)
			continue
		}

		for _, f := range flights {
			log.Printf("Checking flight %s with departure at %s\n", f.ID, time.Unix(f.Departure, 0).Format(time.Kitchen))
			log.Printf("Current time is %s\n", time.Now().Format(time.Kitchen))
			log.Printf("Is departure after now? %t\n", time.Unix(f.Departure, 0).After(time.Now()))
			if time.Unix(f.Departure, 0).After(time.Now()) {
				log.Printf("Found new flight to track: %s ", f.ID)
				b.addFlight(f)
			}
		}
	}
}

func (b *LogicLoop) trackFlight(ctx context.Context, f shared.Flight) {
	ticker := time.NewTicker(1 * time.Minute)
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
			if err != nil {
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
		return
	}

	// check if the flight departed

	if prev.DepActual == 0 && curr.DepActual != 0 {
		depTime := time.Unix(curr.DepActual, 0).Format(time.Kitchen)
		b.sendAlert(f, "flight_departed", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":airplane_departure: *Flight departed!* :airplane_departure:\nDeparture time: %s", depTime), false, false),
			nil,
			nil,
		))
	}

	// check if flight landed

	if prev.ArrActual == 0 && curr.ArrActual != 0 {
		arrTime := time.Unix(curr.ArrActual, 0).Format(time.Kitchen)
		b.sendAlert(f, "flight_landed", slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf(":airplane_arriving: *Flight landed!* :airplane_arriving:\nArrival time: %s", arrTime), false, false),
			nil,
			nil,
		))
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

	// check if flight was cancelled

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
}
