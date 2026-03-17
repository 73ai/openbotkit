package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/73ai/openbotkit/config"
)

func TestBasicAuth_ValidCredentials(t *testing.T) {
	s := &Server{cfg: &config.Config{
		Auth: &config.AuthConfig{Username: "user", Password: "pass"},
	}}

	handler := s.basicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("user", "pass")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestBasicAuth_InvalidCredentials(t *testing.T) {
	s := &Server{cfg: &config.Config{
		Auth: &config.AuthConfig{Username: "user", Password: "pass"},
	}}

	handler := s.basicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("user", "wrong")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestBasicAuth_MissingCredentials(t *testing.T) {
	s := &Server{cfg: &config.Config{
		Auth: &config.AuthConfig{Username: "user", Password: "pass"},
	}}

	handler := s.basicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected WWW-Authenticate header")
	}
}

func TestBasicAuth_NoAuthConfigured_RejectsRequests(t *testing.T) {
	s := &Server{cfg: &config.Config{}}

	handler := s.basicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when no auth configured, got %d", rec.Code)
	}
}

func TestBasicAuth_EnvOverridesConfig(t *testing.T) {
	s := &Server{cfg: &config.Config{
		Auth: &config.AuthConfig{Username: "config-user", Password: "config-pass"},
	}}
	t.Setenv("OBK_AUTH_USERNAME", "env-user")
	t.Setenv("OBK_AUTH_PASSWORD", "env-pass")

	handler := s.basicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Config credentials should fail
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("config-user", "config-pass")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for config creds when env set, got %d", rec.Code)
	}

	// Env credentials should succeed
	req = httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("env-user", "env-pass")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for env creds, got %d", rec.Code)
	}
}
