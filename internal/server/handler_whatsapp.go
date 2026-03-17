package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

type whatsappSendRequest struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

type whatsappSendResponse struct {
	MessageID string `json:"message_id"`
	Timestamp string `json:"timestamp"`
}

func (s *Server) handleWhatsAppSend(w http.ResponseWriter, r *http.Request) {
	var req whatsappSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.To == "" {
		writeError(w, http.StatusBadRequest, "to field is required")
		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text field is required")
		return
	}

	client, err := wasrc.NewClient(r.Context(), s.cfg.WhatsAppSessionDBPath())
	if err != nil {
		slog.Error("whatsapp handler: create client", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create WhatsApp client")
		return
	}
	defer client.Disconnect()

	if !client.IsAuthenticated() {
		writeError(w, http.StatusPreconditionFailed, "WhatsApp not authenticated")
		return
	}

	db, err := store.Open(store.Config{
		Driver: s.cfg.WhatsApp.Storage.Driver,
		DSN:    s.cfg.WhatsAppDataDSN(),
	})
	if err != nil {
		slog.Error("whatsapp handler: open db", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open database")
		return
	}
	defer db.Close()

	result, err := wasrc.SendText(r.Context(), client, db, wasrc.SendInput{
		ChatJID: req.To,
		Text:    req.Text,
	})
	if err != nil {
		slog.Error("whatsapp handler: send", "to", req.To, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to send message")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(whatsappSendResponse{
		MessageID: result.MessageID,
		Timestamp: result.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
	})
}
