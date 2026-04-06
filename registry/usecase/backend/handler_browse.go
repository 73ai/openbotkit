package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	result, err := s.store.ListUseCases(UseCaseFilter{
		Query:    q.Get("q"),
		Domain:   q.Get("domain"),
		Industry: q.Get("industry"),
		Risk:     q.Get("risk"),
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		slog.Error("handler: browse", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list use cases")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleGetUseCase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	uc, err := s.store.GetUseCase(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "use case not found")
		return
	}

	if uc.Visibility == "private" || uc.Status == "draft" {
		callerID := s.optionalUserID(r)
		if callerID != uc.AuthorID {
			writeError(w, http.StatusNotFound, "use case not found")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uc)
}
