package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type dbRequest struct {
	SQL string `json:"sql"`
}

type dbResponse struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

func (s *Server) handleDB(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source")

	var req dbRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SQL == "" {
		writeError(w, http.StatusBadRequest, "sql field is required")
		return
	}

	trimmed := strings.TrimSpace(strings.ToUpper(req.SQL))
	if !strings.HasPrefix(trimmed, "SELECT") {
		writeError(w, http.StatusBadRequest, "only SELECT queries are allowed")
		return
	}

	dsn, err := s.cfg.SourceDataDSN(source)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := os.Stat(dsn); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("database not found: %s", dsn))
		return
	}

	sqlite3, err := exec.LookPath("sqlite3")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sqlite3 not found on server")
		return
	}

	cmd := exec.CommandContext(r.Context(), sqlite3, "-header", "-separator", "\t", dsn, req.SQL)
	out, err := cmd.Output()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("query error: %v", err))
		return
	}

	resp := parseTabOutput(string(out))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func parseTabOutput(raw string) dbResponse {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return dbResponse{Columns: []string{}, Rows: [][]string{}}
	}

	columns := strings.Split(lines[0], "\t")
	rows := make([][]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		rows = append(rows, strings.Split(line, "\t"))
	}

	return dbResponse{Columns: columns, Rows: rows}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
