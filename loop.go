package main

import (
	"flight-tracker-slack/flights"
	"flight-tracker-slack/shared"
	"log"
	"sync"
	"time"
)

type LogicLoop struct {
	Config   shared.Config
	Registry *TrackerRegistry
}

func NewLogicLoop(config shared.Config) *LogicLoop {
	return &LogicLoop{
		Config: config,
		Registry: &TrackerRegistry{
			active: make(map[string]bool), // Don't forget to make the map!
		},
	}
}

type TrackerRegistry struct {
	mu     sync.Mutex
	active map[string]bool
}

func (r *TrackerRegistry) Set(id string, val bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active[id] = val
}

func (r *TrackerRegistry) IsRunning(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active[id]
}

func (b *LogicLoop) Run() {
	log.Println("Starting logic loop...")

	for {
		flights := b.getActiveFlights()

		for _, f := range flights {
			if !b.Registry.IsRunning(f.ID) {
				b.Registry.Set(f.ID, true)

				go b.trackFlight(f)
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func (b *LogicLoop) getActiveFlights() []shared.FlightTrack {

	// query all the tracked flights from the database

	var tracks []shared.FlightTrack
	query := `SELECT id, slack_channel, is_airborne, scheduled_dep, current_eta, last_periodic_update, is_completed FROM flight_tracks WHERE is_completed = 0`

	rows, err := b.Config.UserDB.Query(query)
	if err != nil {
		log.Println("Error querying tracked flights: " + err.Error())
	}

	defer rows.Close()

	for rows.Next() {
		var id string
		var channel string
		var isAirborne int
		var scheduledDep int64
		var currentETA int64
		var lastUpdate int64
		var isCompleted int

		err := rows.Scan(&id, &channel, &isAirborne, &scheduledDep, &currentETA, &lastUpdate, &isCompleted)
		if err != nil {
			log.Println("Error scanning tracked flight row: " + err.Error())
			continue
		}

		track := shared.MapRowToTrack(id, channel, isAirborne, scheduledDep, currentETA, lastUpdate, isCompleted)
		tracks = append(tracks, track)
	}

	return tracks
}

func (b *LogicLoop) trackFlight(flight shared.FlightTrack) {
	// When the function exits (lands or crashes), set to false
	defer b.Registry.Set(flight.ID, false)

	log.Printf("Starting tracker for %s", flight.ID)

	timer := time.NewTimer(1 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// Fetch and logic...
			flightInfo, err := flights.GetFlightInfo(flight.ID)
			if err != nil {
				log.Printf("Error fetching flight info for %s: %v", flight.ID, err)
				timer.Reset(1 * time.Minute)
				continue
			}

			log.Printf("Fetched flight info for %s: status=%s", flight.ID, flightInfo.GetFirstFlight().Code)

			timer.Reset(1 * time.Minute)
		}
	}
}
