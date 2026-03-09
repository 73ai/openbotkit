package server

import "net/http"

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", s.handleHealth)

	auth := s.basicAuth
	mux.Handle("POST /api/db/{source}", auth(http.HandlerFunc(s.handleDB)))
	mux.Handle("GET /api/memory", auth(http.HandlerFunc(s.handleMemoryList)))
	mux.Handle("POST /api/memory", auth(http.HandlerFunc(s.handleMemoryAdd)))
	mux.Handle("DELETE /api/memory/{id}", auth(http.HandlerFunc(s.handleMemoryDelete)))
	mux.Handle("POST /api/applenotes/push", auth(http.HandlerFunc(s.handleAppleNotesPush)))
}
