package main

import (
	"flight-tracker-slack/shared"
	"log"
)

// bot logic here

type Bot struct {
	Config shared.Config
}

func (b *Bot) RunBot() {
	log.Println("Bot is running with token: " + b.Config.SlackToken)
	// Bot logic here
}
