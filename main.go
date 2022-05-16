package main

import (
	"log"

	"4bit.api/v0/cmd"
	"4bit.api/v0/server/route/telegram"
	dotenv "github.com/joho/godotenv"
)

func main() {
	if err := dotenv.Load(); err != nil {
		log.Fatalf("failed to load .env file: %v", err)
	}

	// Initialize telegram bot.
	if err := telegram.Init(); err != nil {
		log.Fatalf("failed to instantiate the telegram bot: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
