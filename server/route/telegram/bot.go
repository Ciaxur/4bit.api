package telegram

import (
	"fmt"
	"log"
	"strings"

	"4bit.api/v0/pkg/camera"
	"4bit.api/v0/server/route/parking"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotCommand struct {
	MethodHandler func(*tgbotapi.Message) tgbotapi.Chattable
}

var (
	BOT_IS_RUNNING bool = false
	BotCommandMp   map[string]BotCommand
)

// Sets up the BotCommandMap with supported commands.
func setupCommands() error {
	if len(BotCommandMp) > 0 {
		return fmt.Errorf("bot command map was already configured")
	}

	BotCommandMp = map[string]BotCommand{
		"help": {
			MethodHandler: func(msg *tgbotapi.Message) tgbotapi.Chattable {
				helpMessage := "Bot Commands are prefixed with '/'. Supported Commands:\n"
				helpMessage += "/help - Prints help menu\n"
				helpMessage += "/parking - Prints the last known altitude of vehicle\n"
				helpMessage += "/snap - Takes snapshot of existing cameras"
				return tgbotapi.NewMessage(msg.Chat.ID, helpMessage)
			},
		},
		"parking": {
			MethodHandler: func(msg *tgbotapi.Message) tgbotapi.Chattable {
				// TODO: Handle various NodeIDs.
				nodeId := uint64(1)
				barEntry, err := parking.GetLastKnownBarometerEntry(nodeId)
				if err != nil {
					return tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("internal failure: %v", err))
				}

				// Map the altitude to a known parking floor.
				parkingFloor, err := parking.GetParkingFloor(barEntry.Altitude)
				parkingFloorMessage := ""
				if err != nil {
					parkingFloor = 0
					parkingFloorMessage = fmt.Sprintf(" - %v", err)
				}

				// Construct nice message.
				replyMsg := "%v:\n"
				replyMsg += "  Entry ID: %d\n"
				replyMsg += "  Node ID: %d\n"
				replyMsg += "  Altitude: %.2f\n"
				replyMsg += "  Floor: %d%s"

				return tgbotapi.NewMessage(
					msg.Chat.ID,
					fmt.Sprintf(
						replyMsg,
						barEntry.Timestamp,
						barEntry.Id,
						nodeId,
						barEntry.Altitude,
						parkingFloor,
						parkingFloorMessage,
					),
				)
			},
		},
		"snap": {
			MethodHandler: func(msg *tgbotapi.Message) tgbotapi.Chattable {
				camPoller := camera.CameraPollerInstance
				images := []interface{}{}

				for _, entry := range camPoller.PollWorkers {
					image := tgbotapi.NewInputMediaPhoto(tgbotapi.FileBytes{
						Name:  entry.Name,
						Bytes: entry.GetLastImage(),
					})
					images = append(images, image)
				}
				BOT.Send(tgbotapi.NewMediaGroup(msg.Chat.ID, images))
				return tgbotapi.NewMessage(
					msg.Chat.ID,
					"Done.",
				)
			},
		},
	}

	return nil
}

func StartBot() {
	// Ensure a single instance of the bot is running.
	if BOT_IS_RUNNING {
		log.Printf("Bot '%s' is already running\n", BOT.Self.UserName)
		return
	}
	BOT_IS_RUNNING = true
	log.Printf("Starting Bot '%s'\n", BOT.Self.UserName)

	if err := setupCommands(); err != nil {
		log.Printf("Failed to setup bot commands: %v\n", err)
	}

	// Listen.
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := BOT.GetUpdatesChan(u)
	for update := range updates {
		if update.Message != nil {
			log.Printf("[+] Bot '%s' New Message from '%s': %s\n", BOT.Self.UserName, update.Message.From.String(), update.Message.Text)

			// Handle supported commands.
			if strings.HasPrefix(update.Message.Text, "/") {
				userCmd := strings.TrimSpace(update.Message.Text[1:])
				log.Printf("[+] Handling user command '%s'\n", userCmd)

				// Obtain the respective bot command handler.
				if botCmd, ok := BotCommandMp[userCmd]; ok {
					botReplyMsg := botCmd.MethodHandler(update.Message)
					BOT.Send(botReplyMsg)
					continue
				}

				// Unknown command.
				helpReply := BotCommandMp["help"].MethodHandler(update.Message)
				BOT.Send(tgbotapi.NewMessage(
					update.Message.Chat.ID,
					fmt.Sprintf("Unknown command '%s'", userCmd),
				))
				BOT.Send(helpReply)
			}
		}
	}
}
