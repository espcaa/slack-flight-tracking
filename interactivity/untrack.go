package interactivity

import (
	"flight-tracker-slack/shared"
	"fmt"
	"log"
	"strings"

	"github.com/slack-go/slack"
)

var UntrackInteraction = shared.Interaction{
	Prefix:  "untrack",
	Execute: HandleUntrackInteraction,
}

func HandleUntrackInteraction(payload slack.InteractionCallback, config shared.Config) {
	log.Printf("Handling untrack interaction for user %s in channel %s\n", payload.User.ID, payload.Channel.ID)
	args := strings.Split(payload.ActionCallback.BlockActions[0].ActionID, "-")
	channel := args[2]
	flightNumber := args[1]
	user := payload.User.ID

	flight, err := shared.FindFlight(flightNumber, channel, user, config)
	if err != nil {
		config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(shared.NewErrorBlocks(err)...))
		return
	}

	err = shared.UntrackFlight(flight.ID, config)
	if err != nil {
		config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(shared.NewErrorBlocks(err)...))
		return
	}

	config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Successfully untracked flight *%s* in channel <#%s>.", flightNumber, channel), false, false),
			nil,
			nil,
		),
	))

}
