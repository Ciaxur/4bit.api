package telegram

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/mux"
)

type TelegramMessageRequest struct {
	ChatID  int64  `json:"chatId"`
	Message string `json:"message"`
	Image   string `json:"image"`
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to parse request body", http.StatusInternalServerError)
		return
	}

	var tlgmMsg TelegramMessageRequest
	if err := json.Unmarshal(body, &tlgmMsg); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Verify request body is valid.
	if tlgmMsg.Message == "" {
		http.Error(w, "message cannot be empty", http.StatusBadRequest)
		return
	}

	if tlgmMsg.ChatID == 0 {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	// Construct and invoke telegram message based on the supplied chat id.
	msg := tgbotapi.NewMessage(tlgmMsg.ChatID, tlgmMsg.Message)
	if _, err := BOT.Send(msg); err != nil {
		http.Error(w, "failed to send message", http.StatusBadRequest)
		return
	}

	// Decode, construct, and upload a given image to telegram.
	if tlgmMsg.Image == "" {
		return
	}
	img, err := base64.StdEncoding.DecodeString(tlgmMsg.Image)
	if err != nil {
		http.Error(w, "failed to decode image", http.StatusBadRequest)
		return
	}

	photoFileBytes := tgbotapi.FileBytes{
		Name:  "image",
		Bytes: img,
	}
	if _, err := BOT.Send(tgbotapi.NewPhoto(tlgmMsg.ChatID, photoFileBytes)); err != nil {
		http.Error(w, "failed to upload the image", http.StatusBadRequest)
		return
	}
}

func CreateRoute(ctx *context.Context, r *mux.Router) {
	r.HandleFunc("", rootHandler).Methods("GET")
	r.HandleFunc("/message", messageHandler).Methods("POST")
}
