package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (s *Server) handleForkUseCase(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	originalID := r.PathValue("id")

	newID := newID()
	newSlug := "fork-" + newID[:12]

	fork, err := s.store.ForkUseCase(originalID, newID, newSlug, userID)
	if err != nil {
		slog.Error("handler: fork use case", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fork use case")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(fork)
}
