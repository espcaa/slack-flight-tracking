package commands

import (
	"flight-tracker-slack/shared"

	"github.com/google/shlex"
	"github.com/slack-go/slack"
)

var HelpCommand = shared.Command{
	Name:        "flights-help",
	Description: "Show help information",
	Usage:       "/help [command_name (optional)]",
	Execute:     Help,
}

func Help(slashCommand slack.SlashCommand, config shared.Config) ([]slack.Block, bool, func()) {
	var specificCommand *shared.Command
	var helpBlocks []slack.Block

	args, err := shlex.Split(slashCommand.Text)
	if err == nil && len(args) >= 1 {
		commandName := args[0]
		for _, cmd := range CommandList {
			if cmd.Name == commandName {
				specificCommand = &cmd
				break
			}
		}
	}

	if specificCommand != nil {
		helpText := "*/" + specificCommand.Name + "*\n" +
			specificCommand.Description + "\n" +
			"*Usage:* `" + specificCommand.Usage + "`"
		helpBlocks = append(helpBlocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, helpText, false, false),
			nil,
			nil,
		))
	} else {
		helpText := "*Available Commands:*\n"
		for _, cmd := range CommandList {
			helpText += "• */" + cmd.Name + "*: " + cmd.Description + "\n"
		}
		helpText += "\nType `/help [command_name]` for detailed info on a specific command."
		helpBlocks = append(helpBlocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, helpText, false, false),
			nil,
			nil,
		))
	}

	return helpBlocks, false, nil
}
