package commands

import (
	"encoding/json"
	"flight-tracker-slack/shared"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

func HandleCommand(name string, w http.ResponseWriter, r *http.Request, config shared.Config) {
	s, err := slack.SlashCommandParse(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	params := &slack.Msg{
		ResponseType: slack.ResponseTypeEphemeral,
		Text:         "processing /" + name + " ...",
	}

	b, err := json.Marshal(params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)

	go func(cmd slack.SlashCommand) {
		var blocks []slack.Block

		switch name {
		case "track":
			blocks = Track(cmd.Text, cmd.ResponseURL, config)
		case "help":
			blocks = Help()
		case "list":
			blocks = List(cmd.Text, cmd.ResponseURL, config)
		case "untrack":
			blocks = Untrack(cmd.Text, cmd.ResponseURL, config)
		default:
			log.Println("Unknown command:", name)
			return
		}

		payload := &slack.WebhookMessage{
			Blocks: &slack.Blocks{
				BlockSet: blocks,
			},
		}

		err := slack.PostWebhook(cmd.ResponseURL, payload)
		if err != nil {
			log.Printf("failed to post to webhook: %v", err)
		}
	}(s)
}
