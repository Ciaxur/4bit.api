package telegram

import (
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	TOKEN string
	BOT   *tgbotapi.BotAPI
)

func Init() error {
	// Extract telegram bot's token from .env file.
	TOKEN = os.Getenv("TELEGRAM_TOKEN")
	if TOKEN == "" {
		return fmt.Errorf("extracted TELEGRAM_TOKEN from .env cannot be empty")
	}

	// Create a telegram bot api instance.
	var err error
	BOT, err = tgbotapi.NewBotAPI(TOKEN)
	if err != nil {
		return fmt.Errorf("failed to create a new bot api: %v", err)
	}
	log.Printf("Authorized on account %s", BOT.Self.UserName)

	return nil
}
