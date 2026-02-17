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
	if len(args) < 3 {
		log.Printf("Invalid action ID format: %s\n", payload.ActionCallback.BlockActions[0].ActionID)
		config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(shared.NewErrorBlocks(fmt.Errorf("Something went wrong while trying to untrack the flight. Please try again."))...))
		return
	}
	channel := args[2]
	flightNumber := args[1]
	user := payload.User.ID

	filter := shared.FlightFilter{
		FlightNumber: flightNumber,
		SlackChannel: channel,
		SlackUserID:  user,
	}

	flight, err := shared.GetFlights(filter, config)
	if err != nil {
		config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(shared.NewErrorBlocks(err)...))
		return
	}
	if len(flight) == 0 {
		config.SlackClient.PostEphemeral(payload.Channel.ID, payload.User.ID, slack.MsgOptionBlocks(shared.NewErrorBlocks(fmt.Errorf("No tracked flight found for flight number %s in channel <#%s>.", flightNumber, channel))...))
		return
	}

	err = shared.UntrackFlight(flight[0].ID, config)
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
