package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func (s *Server) oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.GoogleClientID,
		ClientSecret: s.cfg.GoogleClientSecret,
		RedirectURL:  s.cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func (s *Server) issueJWT(userID, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *Server) setTokenCookie(w http.ResponseWriter, tokenStr string) {
	secure := !s.cfg.DemoLogin
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
	})
}

func (s *Server) handleAuthGoogle(w http.ResponseWriter, r *http.Request) {
	state := randomHex(16)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	url := s.oauthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

func (s *Server) handleAuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		writeError(w, http.StatusBadRequest, "invalid oauth state")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	token, err := s.oauthConfig().Exchange(r.Context(), code)
	if err != nil {
		slog.Error("auth: exchange code", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to exchange code")
		return
	}

	userInfo, err := fetchGoogleUserInfo(r.Context(), token.AccessToken)
	if err != nil {
		slog.Error("auth: fetch user info", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch user info")
		return
	}

	user, err := s.store.UpsertUser(userInfo.ID, userInfo.Email, userInfo.Name, userInfo.Picture)
	if err != nil {
		slog.Error("auth: upsert user", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	jwtStr, err := s.issueJWT(user.ID, user.Email)
	if err != nil {
		slog.Error("auth: issue jwt", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	s.setTokenCookie(w, jwtStr)

	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, s.cfg.FrontendURL+"/dashboard", http.StatusFound)
}

func (s *Server) handleAuthDemo(w http.ResponseWriter, r *http.Request) {
	user, err := s.store.UpsertUser("demo-user-id", "demo@example.com", "Demo User", "")
	if err != nil {
		slog.Error("auth: demo upsert", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create demo user")
		return
	}

	jwtStr, err := s.issueJWT(user.ID, user.Email)
	if err != nil {
		slog.Error("auth: demo jwt", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	s.setTokenCookie(w, jwtStr)
	http.Redirect(w, r, s.cfg.FrontendURL+"/dashboard", http.StatusFound)
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (any, error) {
		return []byte(s.cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid claims")
		return
	}

	userID, _ := claims["sub"].(string)
	user, err := s.store.GetUser(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func fetchGoogleUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo returned %d", resp.StatusCode)
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}
	return &info, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
