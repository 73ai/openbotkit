package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	historysrc "github.com/73ai/openbotkit/service/history"
	"github.com/73ai/openbotkit/service/memory"
)

type memoryAddRequest struct {
	Content  string `json:"content"`
	Category string `json:"category"`
	Source   string `json:"source"`
}

type memoryAddResponse struct {
	ID int64 `json:"id"`
}

type memoryItem struct {
	ID       int64  `json:"id"`
	Content  string `json:"content"`
	Category string `json:"category"`
	Source   string `json:"source"`
}

func (s *Server) openMemoryStore() *memory.Store {
	dir := config.UserMemoryDir()
	memory.EnsureDir(dir)
	return memory.NewStore(dir)
}

func (s *Server) handleMemoryList(w http.ResponseWriter, r *http.Request) {
	ms := s.openMemoryStore()
	category := r.URL.Query().Get("category")

	var (
		memories []memory.Memory
		err      error
	)
	if category != "" {
		memories, err = ms.ListByCategory(memory.Category(category))
	} else {
		memories, err = ms.List()
	}
	if err != nil {
		slog.Error("memory handler: list", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list memories")
		return
	}

	items := make([]memoryItem, len(memories))
	for i, m := range memories {
		items[i] = memoryItem{
			ID:       m.ID,
			Content:  m.Content,
			Category: string(m.Category),
			Source:   m.Source,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) handleMemoryAdd(w http.ResponseWriter, r *http.Request) {
	var req memoryAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.Category == "" {
		req.Category = "preference"
	}
	if req.Source == "" {
		req.Source = "manual"
	}

	ms := s.openMemoryStore()
	id, err := ms.Add(req.Content, memory.Category(req.Category), req.Source, "")
	if err != nil {
		slog.Error("memory handler: add", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to add memory")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(memoryAddResponse{ID: id})
}

func (s *Server) handleMemoryDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid memory ID")
		return
	}

	ms := s.openMemoryStore()
	if err := ms.Delete(id); err != nil {
		slog.Error("memory handler: delete", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete memory")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type memoryExtractRequest struct {
	Last int `json:"last"`
}

type memoryExtractResponse struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
	Skipped int `json:"skipped"`
}

func (s *Server) handleMemoryExtract(w http.ResponseWriter, r *http.Request) {
	var req memoryExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Last <= 0 {
		req.Last = 1
	}

	histDir := config.HistoryDir()
	historysrc.EnsureDir(histDir)
	histStore := historysrc.NewStore(histDir)

	messages, err := histStore.LoadRecentUserMessages(req.Last)
	if err != nil {
		slog.Error("memory extract handler: load messages", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}

	if len(messages) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(memoryExtractResponse{})
		return
	}

	ms := s.openMemoryStore()

	llm, err := s.buildLLM()
	if err != nil {
		slog.Error("memory extract handler: build LLM", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to initialize LLM")
		return
	}

	ctx := r.Context()
	candidates, err := memory.Extract(ctx, llm, messages)
	if err != nil {
		slog.Error("memory extract handler: extract", "error", err)
		writeError(w, http.StatusInternalServerError, "memory extraction failed")
		return
	}

	if len(candidates) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(memoryExtractResponse{})
		return
	}

	result, err := memory.Reconcile(ctx, ms, llm, candidates)
	if err != nil {
		slog.Error("memory extract handler: reconcile", "error", err)
		writeError(w, http.StatusInternalServerError, "memory reconciliation failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memoryExtractResponse{
		Added:   result.Added,
		Updated: result.Updated,
		Deleted: result.Deleted,
		Skipped: result.Skipped,
	})
}

func (s *Server) buildLLM() (memory.LLM, error) {
	registry, err := provider.NewRegistry(s.cfg.Models)
	if err != nil {
		return nil, fmt.Errorf("create provider registry: %w", err)
	}
	router := provider.NewRouter(registry, s.cfg.Models)
	return &memory.RouterLLM{Router: router, Tier: provider.TierFast}, nil
}
