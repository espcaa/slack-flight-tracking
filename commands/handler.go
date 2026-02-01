package commands

import (
	"flight-tracker-slack/shared"
	"fmt"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

func HandleCommand(name string, w http.ResponseWriter, r *http.Request, config shared.Config) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := r.ParseForm()
	if err != nil {
		fmt.Println("Error parsing form:", err)
		return
	}
	commandText := r.FormValue("text")
	responseURL := r.FormValue("response_url")
	var response []slack.Block
	switch name {
	case "track":
		response = Track(commandText, responseURL, config)
	case "help":
		response = Help()
	case "list":
		response = List(commandText, responseURL, config)
	case "untrack":
		response = Untrack(commandText, responseURL, config)
	default:
		log.Println("Unknown command:", name)
		http.Error(w, "Unknown command", http.StatusBadRequest)
	}

	// answer the websocket or smth

	payload := &slack.WebhookMessage{
		Blocks: &slack.Blocks{
			BlockSet: response,
		},
	}

	slack.PostWebhook(responseURL, payload)

	// answer the initial request
	w.Write([]byte("Processing your command..."))
}
