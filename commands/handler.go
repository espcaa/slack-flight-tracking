package commands

import (
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
		InfoCommand,
	}
}

func HandleCommand(name string, w http.ResponseWriter, r *http.Request, config shared.Config) {
	s, err := slack.SlashCommandParse(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	go func(cmd slack.SlashCommand) {
		var blocks []slack.Block
		var in_channel bool = true
		var after func() error = nil

		for _, command := range CommandList {
			if command.Name == name {
				blocks, in_channel, after = command.Execute(s, config)
				break
			}
		}

		var response_type = slack.ResponseTypeInChannel
		if !in_channel {
			response_type = slack.ResponseTypeEphemeral
		}

		payload := &slack.WebhookMessage{
			ResponseType: response_type,
			Blocks: &slack.Blocks{
				BlockSet: blocks,
			},
		}

		err := slack.PostWebhook(cmd.ResponseURL, payload)
		if err != nil {
			log.Printf("failed to post to webhook: %v", err)
		}

		if after != nil {
			after()
		}
	}(s)
}
