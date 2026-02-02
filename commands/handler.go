package commands

import (
	"encoding/json"
	"flight-tracker-slack/shared"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

var CommandList []shared.Command

func init() {
	CommandList = []shared.Command{
		ListCommand,
		TrackCommand,
		UntrackCommand,
		HelpCommand,
	}
}

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

		for _, command := range CommandList {
			if command.Name == name {
				blocks = command.Execute(cmd.Text, cmd.ResponseURL, config)
				break
			}
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
