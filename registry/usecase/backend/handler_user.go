package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type updateUserRequest struct {
	OrgName string `json:"org_name"`
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.store.UpdateUserOrg(userID, req.OrgName); err != nil {
		slog.Error("handler: update user", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	user, err := s.store.GetUser(userID)
	if err != nil {
		slog.Error("handler: get user after update", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
