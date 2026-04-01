package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	st := testStore(t)

	cfg := Config{
		JWTSecret:   "test-secret",
		FrontendURL: "http://localhost:3000",
		DemoLogin:   true,
	}

	Seed(st)

	return NewServer(cfg, st)
}

func issueTestJWT(secret, userID string) string {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

func TestHealthHandler(t *testing.T) {
	srv := testServer(t)
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Fatalf("expected ok, got %s", resp["status"])
	}
}

func TestBrowseHandler(t *testing.T) {
	srv := testServer(t)
	req := httptest.NewRequest("GET", "/api/usecases", nil)
	w := httptest.NewRecorder()
	srv.handleBrowse(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result UseCaseListResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Total == 0 {
		t.Fatal("expected seeded use cases")
	}
}

func TestBrowseFilterByDomain(t *testing.T) {
	srv := testServer(t)
	req := httptest.NewRequest("GET", "/api/usecases?domain=Engineering", nil)
	w := httptest.NewRecorder()
	srv.handleBrowse(w, req)

	var result UseCaseListResult
	json.NewDecoder(w.Body).Decode(&result)
	for _, uc := range result.UseCases {
		if uc.Domain != "Engineering" {
			t.Fatalf("expected Engineering domain, got %s", uc.Domain)
		}
	}
}

func TestGetUseCaseHandler(t *testing.T) {
	srv := testServer(t)

	req := httptest.NewRequest("GET", "/api/usecases", nil)
	w := httptest.NewRecorder()
	srv.handleBrowse(w, req)
	var result UseCaseListResult
	json.NewDecoder(w.Body).Decode(&result)

	if len(result.UseCases) == 0 {
		t.Fatal("no use cases to test")
	}
	ucID := result.UseCases[0].ID

	req2 := httptest.NewRequest("GET", "/api/usecases/"+ucID, nil)
	req2.SetPathValue("id", ucID)
	w2 := httptest.NewRecorder()
	srv.handleGetUseCase(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var uc UseCase
	json.NewDecoder(w2.Body).Decode(&uc)
	if uc.ID != ucID {
		t.Fatalf("expected %s, got %s", ucID, uc.ID)
	}
}

func TestAuthMeWithoutCookie(t *testing.T) {
	srv := testServer(t)
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	w := httptest.NewRecorder()
	srv.handleAuthMe(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMeWithValidCookie(t *testing.T) {
	srv := testServer(t)

	token := issueTestJWT("test-secret", "seed-author")
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	srv.handleAuthMe(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var user User
	json.NewDecoder(w.Body).Decode(&user)
	if user.ID != "seed-author" {
		t.Fatalf("expected seed-author, got %s", user.ID)
	}
}

func TestCreateUseCaseRequiresAuth(t *testing.T) {
	srv := testServer(t)

	mux := http.NewServeMux()
	srv.routes(mux)

	req := httptest.NewRequest("POST", "/api/usecases", strings.NewReader(`{"title":"Test"}`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateAndDeleteUseCase(t *testing.T) {
	srv := testServer(t)

	token := issueTestJWT("test-secret", "seed-author")

	body := `{"title":"New UC","description":"A new use case","domain":"Sales","risk_level":"low","roi_potential":"high"}`
	req := httptest.NewRequest("POST", "/api/usecases", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	req.Header.Set("Content-Type", "application/json")

	mux := http.NewServeMux()
	srv.routes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created UseCase
	json.NewDecoder(w.Body).Decode(&created)

	req2 := httptest.NewRequest("DELETE", "/api/usecases/"+created.ID, nil)
	req2.SetPathValue("id", created.ID)
	req2.AddCookie(&http.Cookie{Name: "token", Value: token})
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != 204 {
		t.Fatalf("expected 204, got %d: %s", w2.Code, w2.Body.String())
	}
}
