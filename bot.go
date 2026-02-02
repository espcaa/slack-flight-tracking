package main

import (
	"flight-tracker-slack/shared"
	"log"
)

type Bot struct {
	Config shared.Config
}

func (b *Bot) RunBot() {
	log.Println("Bot is runningggg")
}
