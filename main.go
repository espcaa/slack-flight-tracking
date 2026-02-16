package main

import (
	"database/sql"
	"flight-tracker-slack/commands"
	"flight-tracker-slack/flights"
	"flight-tracker-slack/interactivity"
	"flight-tracker-slack/maps"
	"flight-tracker-slack/shared"
	"image/jpeg"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	_ "modernc.org/sqlite"
)

func main() {

	log.Println("Starting...")

	err := godotenv.Load()
	if err != nil {
		log.Println("error loading .env file or file not found")
	} else {
		log.Println(".env file loaded")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	} else {
		log.Println("running on port: " + port)
	}
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable not set")
	}

	slackSigningSecret := os.Getenv("SLACK_SIGNING_SECRET")
	if slackSigningSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable not set")
	}

	tileStore := maps.NewTileStore("./data/map")

	config := shared.Config{
		Port:          port,
		SlackClient:   slack.New(slackToken),
		SlackToken:    slackToken,
		TileStore:     tileStore,
		SigningSecret: slackSigningSecret,
	}

	Start(config)
}

func Start(config shared.Config) {

	db, err := sql.Open("sqlite", "file:data/userdata.db?mode=rwc")
	if err != nil {
		log.Fatal("Error opening flights database: " + err.Error())
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("DB ping failed: " + err.Error())
	} else {
		log.Println("Connected to the db!")
	}

	config.UserDB = db
	setupDatabase(db)

	r := chi.NewRouter()

	r.Post("/commands/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		log.Println("Received command: " + name)
		commands.HandleCommand(name, w, r, config)
	})

	// interactivity

	r.Post("/slack/interactivity", func(w http.ResponseWriter, r *http.Request) {
		interactivity.HandleInteraction(w, r, config)
	})

	r.Get("/commands/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("seems like you're trying to run the " + name + " command..."))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("haiii"))
	})

	r.Get("/map/{flightID}", func(w http.ResponseWriter, r *http.Request) {
		flightID := chi.URLParam(r, "flightID")

		flightDetails, err := flights.GetFlightInfo(flightID)
		if err != nil {
			http.Error(w, "Flight not found", http.StatusNotFound)
			return
		}

		var flight flights.FlightDetail
		for _, f := range flightDetails.Flights {
			flight = f
			break
		}

		img, err := maps.GenerateMapFromFlightDetail(config.TileStore, flight)
		if err != nil {
			http.Error(w, "Failed to generate map", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=300")

		err = jpeg.Encode(w, img, &jpeg.Options{Quality: 80})
		if err != nil {
			log.Println("Error encoding JPEG:", err)
		}
	})

	log.Println("Starting server on port " + config.Port)

	LogicLoop := NewLogicLoop(config)
	go LogicLoop.Run()
	err = http.ListenAndServe(":"+config.Port, r)
	if err != nil {
		log.Fatal("Error starting server: " + err.Error())
	}

}

func setupDatabase(db *sql.DB) {
	schema := `
    CREATE TABLE IF NOT EXISTS flights (
        id TEXT PRIMARY KEY,
        flight_number TEXT,
        slack_channel TEXT,
        slack_user_id TEXT,
        departure INTEGER
    );
    CREATE TABLE IF NOT EXISTS alerts_sent (
        flight_id TEXT,
        alert_type TEXT,
        PRIMARY KEY (flight_id, alert_type)
    );
    CREATE TABLE IF NOT EXISTS flight_state (
        flight_id TEXT PRIMARY KEY,
        status TEXT,
        origin_gate TEXT,
        origin_terminal TEXT,
        dest_gate TEXT,
        dest_terminal TEXT,
        dep_scheduled INTEGER,
        dep_estimated INTEGER,
        dep_actual INTEGER,
        takeoff_actual INTEGER,
        takeoff_estimated INTEGER,
        landing_actual INTEGER,
        landing_estimated INTEGER,
        arr_scheduled INTEGER,
        arr_estimated INTEGER,
        arr_actual INTEGER,
        altitude INTEGER,
        groundspeed INTEGER,
        updated_at INTEGER
    );
    `

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatal("Could not create tables:", err)
	}
}
