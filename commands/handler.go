package commands

import (
	"bytes"
	"encoding/json"
	"flight-tracker-slack/shared"
	"io"
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

	// verify the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sv, err := slack.NewSecretsVerifier(r.Header, config.SigningSecret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))

	w.WriteHeader(http.StatusOK)

	// parse the slash command
	s, err := slack.SlashCommandParse(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// run the command in a separate goroutine
	go func(cmd slack.SlashCommand) {
		var blocks []slack.Block
		var inChannel bool = true
		var after func(responseURL string) error = nil

		for _, command := range CommandList {
			if command.Name == name {
				blocks, inChannel, after = command.Execute(s, config)
				break
			}
		}

		var responseType = slack.ResponseTypeInChannel
		if !inChannel {
			responseType = slack.ResponseTypeEphemeral
		}

		// print the blocks in json for debugging
		b, _ := json.MarshalIndent(blocks, "", "  ")
		log.Printf("Response blocks: %s", string(b))

		payload := &slack.WebhookMessage{
			ResponseType: responseType,
			Blocks: &slack.Blocks{
				BlockSet: blocks,
			},
		}

		err := slack.PostWebhook(cmd.ResponseURL, payload)
		if err != nil {
			log.Printf("failed to post to webhook: %v", err)
			slack.PostWebhook(cmd.ResponseURL, &slack.WebhookMessage{
				ResponseType: slack.ResponseTypeEphemeral,
				Blocks: &slack.Blocks{
					BlockSet: NewErrorBlocks(err),
				},
			})
		}

		if after != nil {
			err = after(cmd.ResponseURL)
			if err != nil {
				log.Printf("after function error: %v", err)
				err = slack.PostWebhook(cmd.ResponseURL, &slack.WebhookMessage{
					ResponseType: slack.ResponseTypeEphemeral,
					Blocks: &slack.Blocks{
						BlockSet: []slack.Block{
							slack.NewSectionBlock(
								slack.NewTextBlockObject(slack.MarkdownType, "Failed to execute the command entirely :x:", false, false),
								nil,
								nil,
							),
							slack.NewSectionBlock(
								slack.NewTextBlockObject(slack.MarkdownType, "_Error details:_ ```"+err.Error()+"```", false, false),
								nil,
								nil,
							),
						},
					},
				})
				if err != nil {
					slack.PostWebhook(cmd.ResponseURL, &slack.WebhookMessage{
						ResponseType: slack.ResponseTypeEphemeral,
						Blocks: &slack.Blocks{
							BlockSet: NewErrorBlocks(err),
						},
					})
					log.Printf("failed to post error message to webhook: %v", err)
				}
			}
		}
	}(s)
}
