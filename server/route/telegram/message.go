package telegram

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/mux"
)

type TelegramMessageRequest struct {
	ChatID  int64  `json:"chatId"`
	Message string `json:"message"`

	// Base64-encoded image.
	Image string `json:"image"`
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("telegram/message: failed to parse request body -> %v", err)
		http.Error(w, "failed to parse request body", http.StatusInternalServerError)
		return
	}

	var tlgmMsg TelegramMessageRequest
	if err := json.Unmarshal(body, &tlgmMsg); err != nil {
		log.Printf("telegram/message: invalid request body -> %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Verify request body is valid.
	if tlgmMsg.Message == "" {
		log.Printf("telegram/message: message cannot be empty -> %v", err)
		http.Error(w, "message cannot be empty", http.StatusBadRequest)
		return
	}

	if tlgmMsg.ChatID == 0 {
		log.Printf("telegram/message: invalid chat id -> %v", err)
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	// Construct and invoke telegram message based on the supplied chat id.
	msg := tgbotapi.NewMessage(tlgmMsg.ChatID, tlgmMsg.Message)
	if _, err := BOT.Send(msg); err != nil {
		log.Printf("telegram/message: failed to send message -> %v", err)
		http.Error(w, "failed to send message", http.StatusBadRequest)
		return
	}

	// Decode, construct, and upload a given image to telegram.
	if len(tlgmMsg.Image) == 0 {
		log.Printf("telegram/message: Skipping image. No image in request to upload")
		return
	}
	img, err := base64.StdEncoding.DecodeString(tlgmMsg.Image)
	if err != nil {
		log.Printf("telegram/message: base64-encoded image expected, failed to decode -> %v", err)
		http.Error(w, "failed to decode image", http.StatusBadRequest)
		return
	}

	photoFileBytes := tgbotapi.FileBytes{
		Name:  "image",
		Bytes: img,
	}
	if _, err := BOT.Send(tgbotapi.NewPhoto(tlgmMsg.ChatID, photoFileBytes)); err != nil {
		log.Printf("telegram/message: failed to upload the image -> %v", err)
		http.Error(w, "failed to upload the image", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func CreateRoute(ctx *context.Context, r *mux.Router) {
	r.HandleFunc("", rootHandler).Methods("GET")
	r.HandleFunc("/message", messageHandler).Methods("POST")
}
