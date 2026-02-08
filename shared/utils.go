package shared

import (
	"fmt"

	"github.com/slack-go/slack"
)

func NewErrorBlocks(err error, customMessage ...string) []slack.Block {
	message := "Something wrong happened :x:"
	if len(customMessage) > 0 && customMessage[0] != "" {
		message = customMessage[0]
	}

	return []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, message, false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("_Error details:_ ```%v```", err), false, false),
			nil,
			nil,
		),
	}
}
