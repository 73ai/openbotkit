package server

import "net/http"

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", s.handleHealth)

	auth := s.basicAuth
	mux.Handle("POST /api/db/{source}", auth(http.HandlerFunc(s.handleDB)))
	mux.Handle("GET /api/memory", auth(http.HandlerFunc(s.handleMemoryList)))
	mux.Handle("POST /api/memory", auth(http.HandlerFunc(s.handleMemoryAdd)))
	mux.Handle("DELETE /api/memory/{id}", auth(http.HandlerFunc(s.handleMemoryDelete)))
	mux.Handle("POST /api/memory/extract", auth(http.HandlerFunc(s.handleMemoryExtract)))
	mux.Handle("POST /api/applenotes/push", auth(http.HandlerFunc(s.handleAppleNotesPush)))
	mux.Handle("POST /api/imessage/push", auth(http.HandlerFunc(s.handleIMessagePush)))
	mux.Handle("POST /api/gmail/send", auth(http.HandlerFunc(s.handleGmailSend)))
	mux.Handle("POST /api/gmail/draft", auth(http.HandlerFunc(s.handleGmailDraft)))
	mux.Handle("POST /api/gmail/sync", auth(http.HandlerFunc(s.handleGmailSync)))
	mux.Handle("POST /api/whatsapp/send", auth(http.HandlerFunc(s.handleWhatsAppSend)))

	mux.HandleFunc("GET /auth/google/callback", s.handleGoogleAuthCallback)

	mux.Handle("GET /auth/whatsapp", auth(http.HandlerFunc(s.handleWhatsAppAuthPage)))
	mux.Handle("GET /auth/whatsapp/api/qr", auth(http.HandlerFunc(s.handleWhatsAppAuthQR)))
}
