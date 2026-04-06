package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())

	usecases, err := s.store.ListUserUseCases(userID)
	if err != nil {
		slog.Error("handler: dashboard", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list your use cases")
		return
	}
	if usecases == nil {
		usecases = []UseCase{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"use_cases": usecases,
	})
}
