package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	imsrc "github.com/73ai/openbotkit/source/imessage"
	"github.com/73ai/openbotkit/store"
)

type imessageItem struct {
	GUID                  string `json:"guid"`
	AppleROWID            int64  `json:"apple_rowid"`
	Text                  string `json:"text"`
	ChatGUID              string `json:"chat_guid"`
	SenderID              string `json:"sender_id"`
	SenderService         string `json:"sender_service"`
	IsFromMe              bool   `json:"is_from_me"`
	IsRead                bool   `json:"is_read"`
	Date                  string `json:"date"`
	DateRead              string `json:"date_read"`
	ReplyToGUID           string `json:"reply_to_guid"`
	AssociatedMessageGUID string `json:"associated_msg_guid"`
	AssociatedMessageType int    `json:"associated_msg_type"`
	AttachmentsJSON       string `json:"attachments_json"`
	ChatDisplayName       string `json:"chat_display_name"`
}

func (s *Server) handleIMessagePush(w http.ResponseWriter, r *http.Request) {
	var messages []imessageItem
	if err := json.NewDecoder(r.Body).Decode(&messages); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	db, err := store.Open(store.Config{
		Driver: s.cfg.IMessage.Storage.Driver,
		DSN:    s.cfg.IMessageDataDSN(),
	})
	if err != nil {
		slog.Error("imessage handler: open db", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open database")
		return
	}
	defer db.Close()

	saved := 0
	for _, m := range messages {
		date, _ := time.Parse("2006-01-02T15:04:05Z07:00", m.Date)
		dateRead, _ := time.Parse("2006-01-02T15:04:05Z07:00", m.DateRead)

		msg := &imsrc.Message{
			GUID:                  m.GUID,
			AppleROWID:            m.AppleROWID,
			Text:                  m.Text,
			ChatGUID:              m.ChatGUID,
			SenderID:              m.SenderID,
			SenderService:         m.SenderService,
			IsFromMe:              m.IsFromMe,
			IsRead:                m.IsRead,
			Date:                  date,
			DateRead:              dateRead,
			ReplyToGUID:           m.ReplyToGUID,
			AssociatedMessageGUID: m.AssociatedMessageGUID,
			AssociatedMessageType: m.AssociatedMessageType,
			AttachmentsJSON:       m.AttachmentsJSON,
			ChatDisplayName:       m.ChatDisplayName,
		}
		if err := imsrc.SaveMessage(db, msg); err != nil {
			slog.Error("imessage handler: save message", "guid", m.GUID, "error", err)
			writeError(w, http.StatusInternalServerError, "failed to save message")
			return
		}
		saved++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"saved": saved})
}
