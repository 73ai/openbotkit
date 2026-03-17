package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/73ai/openbotkit/memory"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/store"
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
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

func (s *Server) openMemoryDB() (*store.DB, error) {
	db, err := store.Open(store.Config{
		Driver: s.cfg.UserMemory.Storage.Driver,
		DSN:    s.cfg.UserMemoryDataDSN(),
	})
	if err != nil {
		return nil, fmt.Errorf("open memory db: %w", err)
	}
	return db, nil
}

func (s *Server) handleMemoryList(w http.ResponseWriter, r *http.Request) {
	db, err := s.openMemoryDB()
	if err != nil {
		slog.Error("memory handler: open db", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open database")
		return
	}
	defer db.Close()

	category := r.URL.Query().Get("category")

	var memories []memory.Memory
	if category != "" {
		memories, err = memory.ListByCategory(db, memory.Category(category))
	} else {
		memories, err = memory.List(db)
	}
	if err != nil {
		slog.Error("memory handler: list", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list memories")
		return
	}

	items := make([]memoryItem, len(memories))
	for i, m := range memories {
		items[i] = memoryItem{
			ID:        m.ID,
			Content:   m.Content,
			Category:  string(m.Category),
			Source:    m.Source,
			CreatedAt: m.CreatedAt.Format("2006-01-02 15:04:05"),
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

	db, err := s.openMemoryDB()
	if err != nil {
		slog.Error("memory handler: open db", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open database")
		return
	}
	defer db.Close()

	id, err := memory.Add(db, req.Content, memory.Category(req.Category), req.Source, "")
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

	db, err := s.openMemoryDB()
	if err != nil {
		slog.Error("memory handler: open db", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open database")
		return
	}
	defer db.Close()

	if err := memory.Delete(db, id); err != nil {
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

	histDB, err := store.Open(store.Config{
		Driver: s.cfg.History.Storage.Driver,
		DSN:    s.cfg.HistoryDataDSN(),
	})
	if err != nil {
		slog.Error("memory extract handler: open history db", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open history database")
		return
	}
	defer histDB.Close()

	messages, err := loadRecentMessages(histDB, req.Last)
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

	memDB, err := s.openMemoryDB()
	if err != nil {
		slog.Error("memory extract handler: open memory db", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open database")
		return
	}
	defer memDB.Close()

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

	result, err := memory.Reconcile(ctx, memDB, llm, candidates)
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

func loadRecentMessages(db *store.DB, lastN int) ([]string, error) {
	query := db.Rebind(`
		SELECT m.content FROM history_messages m
		JOIN history_conversations c ON c.id = m.conversation_id
		WHERE m.role = 'user'
		ORDER BY c.updated_at DESC, m.timestamp DESC
		LIMIT ?`)

	rows, err := db.Query(query, lastN*50)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, content)
	}
	return messages, rows.Err()
}

func (s *Server) buildLLM() (memory.LLM, error) {
	registry, err := provider.NewRegistry(s.cfg.Models)
	if err != nil {
		return nil, fmt.Errorf("create provider registry: %w", err)
	}
	router := provider.NewRouter(registry, s.cfg.Models)
	return &memory.RouterLLM{Router: router, Tier: provider.TierFast}, nil
}

