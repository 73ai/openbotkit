package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/oauth/google"
	gmailsrc "github.com/73ai/openbotkit/source/gmail"
	"github.com/73ai/openbotkit/store"
)

type gmailSendRequest struct {
	To      []string `json:"to"`
	Cc      []string `json:"cc"`
	Bcc     []string `json:"bcc"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Account string   `json:"account"`
}

type gmailSendResponse struct {
	MessageID string `json:"message_id"`
	ThreadID  string `json:"thread_id"`
}

func (s *Server) handleGmailSend(w http.ResponseWriter, r *http.Request) {
	var req gmailSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.To) == 0 {
		writeError(w, http.StatusBadRequest, "at least one recipient is required")
		return
	}

	g := s.newGmail()
	result, err := g.Send(r.Context(), gmailsrc.ComposeInput{
		To:      req.To,
		Cc:      req.Cc,
		Bcc:     req.Bcc,
		Subject: req.Subject,
		Body:    req.Body,
		Account: req.Account,
	})
	if err != nil {
		slog.Error("gmail handler: send", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to send email")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gmailSendResponse{
		MessageID: result.MessageID,
		ThreadID:  result.ThreadID,
	})
}

type gmailDraftResponse struct {
	DraftID   string `json:"draft_id"`
	MessageID string `json:"message_id"`
	ThreadID  string `json:"thread_id"`
}

func (s *Server) handleGmailDraft(w http.ResponseWriter, r *http.Request) {
	var req gmailSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.To) == 0 {
		writeError(w, http.StatusBadRequest, "at least one recipient is required")
		return
	}

	g := s.newGmail()
	result, err := g.CreateDraft(r.Context(), gmailsrc.ComposeInput{
		To:      req.To,
		Cc:      req.Cc,
		Bcc:     req.Bcc,
		Subject: req.Subject,
		Body:    req.Body,
		Account: req.Account,
	})
	if err != nil {
		slog.Error("gmail handler: create draft", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create draft")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gmailDraftResponse{
		DraftID:   result.DraftID,
		MessageID: result.MessageID,
		ThreadID:  result.ThreadID,
	})
}

type gmailSyncRequest struct {
	Full       bool   `json:"full"`
	After      string `json:"after"`
	Account    string `json:"account"`
	DaysWindow int    `json:"days_window"`
}

type gmailSyncResponse struct {
	Fetched int `json:"fetched"`
	Skipped int `json:"skipped"`
	Errors  int `json:"errors"`
}

func (s *Server) handleGmailSync(w http.ResponseWriter, r *http.Request) {
	var req gmailSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	db, err := store.Open(store.Config{
		Driver: s.cfg.Gmail.Storage.Driver,
		DSN:    s.cfg.GmailDataDSN(),
	})
	if err != nil {
		slog.Error("gmail handler: open db", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open database")
		return
	}
	defer db.Close()

	attachDir := filepath.Join(config.SourceDir("gmail"), "attachments")

	g := s.newGmail()
	result, err := g.Sync(r.Context(), db, gmailsrc.SyncOptions{
		Full:                req.Full,
		After:               req.After,
		Account:             req.Account,
		DaysWindow:          req.DaysWindow,
		DownloadAttachments: s.cfg.Gmail.DownloadAttachments,
		AttachmentsDir:      attachDir,
	})
	if err != nil {
		slog.Error("gmail handler: sync", "error", err)
		writeError(w, http.StatusInternalServerError, "sync failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gmailSyncResponse{
		Fetched: result.Fetched,
		Skipped: result.Skipped,
		Errors:  result.Errors,
	})
}

func (s *Server) newGmail() *gmailsrc.Gmail {
	gp := google.New(google.Config{
		CredentialsFile: s.cfg.GoogleCredentialsFile(),
		TokenDBPath:     s.cfg.GoogleTokenDBPath(),
	})
	return gmailsrc.New(gmailsrc.Config{Provider: gp})
}
