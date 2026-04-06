package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

type useCaseRequest struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	Domain            string `json:"domain"`
	IndustryTags      string `json:"industry_tags"`
	RiskLevel         string `json:"risk_level"`
	ROIPotential      string `json:"roi_potential"`
	Status            string `json:"status"`
	ImplStatus        string `json:"impl_status"`
	Visibility        string `json:"visibility"`
	SafetyPII         bool   `json:"safety_pii"`
	SafetyAutonomous  bool   `json:"safety_autonomous"`
	SafetyBlastRadius string `json:"safety_blast_radius"`
	SafetyOversight   string `json:"safety_oversight"`
}

func (s *Server) handleCreateUseCase(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())

	var req useCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RiskLevel == "" {
		req.RiskLevel = "medium"
	}
	if req.ROIPotential == "" {
		req.ROIPotential = "medium"
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.ImplStatus == "" {
		req.ImplStatus = "evaluating"
	}
	if req.Visibility == "" {
		req.Visibility = "public"
	}

	if err := validateUseCaseRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id := newID()
	uc := &UseCase{
		ID:                id,
		Title:             req.Title,
		Slug:              slugify(req.Title) + "-" + id[:8],
		Description:       req.Description,
		Domain:            req.Domain,
		IndustryTags:      req.IndustryTags,
		RiskLevel:         req.RiskLevel,
		ROIPotential:      req.ROIPotential,
		Status:            req.Status,
		ImplStatus:        req.ImplStatus,
		Visibility:        req.Visibility,
		SafetyPII:         req.SafetyPII,
		SafetyAutonomous:  req.SafetyAutonomous,
		SafetyBlastRadius: req.SafetyBlastRadius,
		SafetyOversight:   req.SafetyOversight,
		AuthorID:          userID,
	}

	if err := s.store.CreateUseCase(uc); err != nil {
		slog.Error("handler: create use case", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create use case")
		return
	}

	created, err := s.store.GetUseCase(id)
	if err != nil {
		slog.Error("handler: get created use case", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch created use case")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *Server) handleUpdateUseCase(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	id := r.PathValue("id")

	var req useCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateUseCaseRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	uc := &UseCase{
		ID:                id,
		Title:             req.Title,
		Slug:              slugify(req.Title) + "-" + id[:8],
		Description:       req.Description,
		Domain:            req.Domain,
		IndustryTags:      req.IndustryTags,
		RiskLevel:         req.RiskLevel,
		ROIPotential:      req.ROIPotential,
		Status:            req.Status,
		ImplStatus:        req.ImplStatus,
		Visibility:        req.Visibility,
		SafetyPII:         req.SafetyPII,
		SafetyAutonomous:  req.SafetyAutonomous,
		SafetyBlastRadius: req.SafetyBlastRadius,
		SafetyOversight:   req.SafetyOversight,
		AuthorID:          userID,
	}

	if err := s.store.UpdateUseCase(uc); err != nil {
		slog.Error("handler: update use case", "error", err)
		writeError(w, http.StatusForbidden, "use case not found or not owned by you")
		return
	}

	updated, err := s.store.GetUseCase(id)
	if err != nil {
		slog.Error("handler: get updated use case", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch updated use case")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (s *Server) handleDeleteUseCase(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	id := r.PathValue("id")

	if err := s.store.DeleteUseCase(id, userID); err != nil {
		slog.Error("handler: delete use case", "error", err)
		writeError(w, http.StatusForbidden, "use case not found or not owned by you")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	slug := strings.ToLower(s)
	slug = nonAlphaNum.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 60 {
		slug = slug[:60]
	}
	return slug
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
