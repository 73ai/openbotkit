package server

import (
	"crypto/subtle"
	"net/http"
	"os"
)

const maxRequestBody = 1 << 20 // 1 MB

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password := s.authCredentials()

		u, p, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(u), []byte(username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(p), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="obk"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authCredentials() (string, string) {
	if u := os.Getenv("OBK_AUTH_USERNAME"); u != "" {
		return u, os.Getenv("OBK_AUTH_PASSWORD")
	}
	if s.cfg.Auth != nil {
		return s.cfg.Auth.Username, s.cfg.Auth.Password
	}
	return "", ""
}
