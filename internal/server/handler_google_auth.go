package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/priyanshujain/openbotkit/internal/skills"
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

	scopes, account, ok := s.scopeWaiter.Lookup(state)
	if !ok {
		account = s.resolveAccount()
	}
	if account == "" {
		http.Error(w, "no Google account configured", http.StatusBadRequest)
		return
	}

	if err := s.google.ExchangeCode(r.Context(), code, account, scopes); err != nil {
		slog.Error("google auth callback: exchange code", "error", err)
		s.scopeWaiter.Signal(state, err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	s.scopeWaiter.Signal(state, nil)

	go func() {
		if err := skills.RefreshGWSSkills(s.cfg); err != nil {
			slog.Warn("gws skill refresh after auth failed", "error", err)
		}
	}()

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
