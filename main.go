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
		log.Fatal("PORT environment variable not set")
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
		TileStore:     tileStore,
		SigningSecret: slackSigningSecret,
	}

	Start(config)
}

func Start(config shared.Config) {

	db, err := sql.Open("sqlite", "./flights.db")
	if err != nil {
		log.Fatal("Error opening flights database: " + err.Error())
	}
	defer db.Close()
	log.Println("Connected to flights database")

	config.UserDB = db

	r := chi.NewRouter()

	r.Post("/commands/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		log.Println("Received command: " + name)
		commands.HandleCommand(name, w, r, config)
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
}
