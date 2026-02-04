package commands

import (
	"fmt"

	"github.com/slack-go/slack"
)

func NewErrorBlocks(err error) []slack.Block {
	return []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, "Something wrong happened :x:", false, false),
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
