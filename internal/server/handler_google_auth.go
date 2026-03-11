package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

func (s *Server) handleGoogleAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}
	if state == "" {
		http.Error(w, "missing state parameter", http.StatusBadRequest)
		return
	}

	// Extract account from state if embedded, otherwise use first known account.
	account := s.resolveAccount()

	// Extract scopes from state prefix (gws-<timestamp>).
	scopes := s.scopesFromState(state)

	if err := s.google.ExchangeCode(r.Context(), code, account, scopes); err != nil {
		slog.Error("google auth callback: exchange code", "error", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	s.scopeWaiter.Signal(state, nil)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "<h1>Access granted!</h1><p>You can close this tab.</p>")
}

func (s *Server) resolveAccount() string {
	accounts, err := s.google.Accounts(s.ctx)
	if err != nil || len(accounts) == 0 {
		return ""
	}
	return accounts[0]
}

func (s *Server) scopesFromState(state string) []string {
	// State is formatted as "gws-<timestamp>" — no embedded scopes.
	// Return nil; ExchangeCode will merge with existing.
	_ = strings.TrimPrefix(state, "gws-")
	return nil
}
