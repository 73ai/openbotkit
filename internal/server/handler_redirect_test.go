package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleAuthRedirect_MissingURL(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest("GET", "/auth/redirect", nil)
	rec := httptest.NewRecorder()
	s.handleAuthRedirect(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAuthRedirect_NonGoogleURL(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest("GET", "/auth/redirect?url="+url.QueryEscape("https://evil.com/phish"), nil)
	rec := httptest.NewRecorder()
	s.handleAuthRedirect(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAuthRedirect_HTTPNotHTTPS(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest("GET", "/auth/redirect?url="+url.QueryEscape("http://accounts.google.com/o/oauth2/auth"), nil)
	rec := httptest.NewRecorder()
	s.handleAuthRedirect(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAuthRedirect_ValidGoogleURL(t *testing.T) {
	s := &Server{}
	googleURL := "https://accounts.google.com/o/oauth2/auth?client_id=test&scope=calendar"
	req := httptest.NewRequest("GET", "/auth/redirect?url="+url.QueryEscape(googleURL), nil)
	rec := httptest.NewRecorder()
	s.handleAuthRedirect(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `target="_blank"`) {
		t.Error("response should contain target=_blank")
	}
	if !strings.Contains(body, `rel="noopener"`) {
		t.Error("response should contain rel=noopener")
	}
	if !strings.Contains(body, "https://accounts.google.com/o/oauth2/auth?client_id=test&amp;scope=calendar") {
		t.Error("response should contain the Google OAuth URL in href")
	}
}

func TestHandleAuthRedirect_HTMLEscapesAmpersand(t *testing.T) {
	s := &Server{}
	googleURL := "https://accounts.google.com/o/oauth2/auth?client_id=test&scope=calendar"
	req := httptest.NewRequest("GET", "/auth/redirect?url="+url.QueryEscape(googleURL), nil)
	rec := httptest.NewRecorder()
	s.handleAuthRedirect(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "client_id=test&amp;scope=calendar") {
		t.Errorf("ampersand should be HTML-escaped in href, got body:\n%s", body)
	}
}
