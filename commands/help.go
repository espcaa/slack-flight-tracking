package commands

import "github.com/slack-go/slack"

func Help() []slack.Block {
	helpText := "*Available commands:*\n" +
		"• `/track [flight_number]` - Track a flight\n" +
		"• `/untrack [flight_number]` - Untrack a flight\n" +
		"• `/list` - List all tracked flights\n" +
		"• `/help` - Show this help message"

	sectionBlock := slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, helpText, false, false),
		nil,
		nil,
	)

	return []slack.Block{
		sectionBlock,
	}
}
