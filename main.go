package main

import (
	"database/sql"
	"flight-tracker-slack/commands"
	"flight-tracker-slack/maps"
	"flight-tracker-slack/shared"
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

	r := chi.NewRouter()

	r.Post("/commands/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		log.Println("Received command: " + name)
		commands.HandleCommand(name, w, r, config)
	})

	// interactivity

	r.Post("/slack/interactivity", func(w http.ResponseWriter, r *http.Request) {
		// print all the data we can get
		r.ParseForm()
		log.Println("Received interactivity payload: " + r.FormValue("payload"))
		// acknowledge receipt
		w.WriteHeader(http.StatusOK)
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

	log.Println("Starting server on port " + config.Port)

	err = http.ListenAndServe(":"+config.Port, r)
	if err != nil {
		log.Fatal("Error starting server: " + err.Error())
	}

	LogicLoop := LogicLoop{
		Config: config,
	}
	go LogicLoop.Run()
}

func setupDatabase(db *sql.DB) {
	schema := `
    CREATE TABLE IF NOT EXISTS flights (
        id TEXT PRIMARY KEY,
        slack_channel TEXT,
        status TEXT,
        is_airborne INTEGER DEFAULT 0,
        scheduled_dep INTEGER,
        current_eta INTEGER,
        last_periodic_update INTEGER DEFAULT 0,
        is_completed INTEGER DEFAULT 0
    );
    CREATE TABLE IF NOT EXISTS alerts_sent (
        flight_id TEXT,
        alert_type TEXT,
        PRIMARY KEY (flight_id, alert_type)
    );`

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatal("Could not create tables:", err)
	}
}
