package main

import (
	"database/sql"
	"flight-tracker-slack/commands"
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
		log.Fatal("Error loading .env file")
	}
	log.Println("Loaded .env file")

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable not set")
	}
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable not set")
	}

	config := shared.Config{
		Port:        port,
		SlackClient: slack.New(slackToken),
	}

	Start(config)
}

func Start(config shared.Config) {

	// load the db

	db, err := sql.Open("sqlite", "./flights.db")
	if err != nil {
		log.Fatal("Error opening flights database: " + err.Error())
	}
	defer db.Close()
	log.Println("Connected to flights database")

	config.UserDB = db

	// start the http server

	r := chi.NewRouter()

	r.Post("/commands/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		log.Println("Received command: " + name)
		commands.HandleCommand(name, w, r, config)
	})

	r.Get("/commands/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		// log.Println("seems like you're trying to run the " + name + " command...")
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

	// commands endpoints

	http.ListenAndServe(":"+config.Port, r)
}
