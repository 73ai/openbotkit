package gmail

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
)

// TokenStore persists OAuth tokens in a SQLite database.
type TokenStore struct {
	db *sql.DB
}

// NewTokenStore opens (or creates) the credentials database and runs migrations.
func NewTokenStore(dbPath string) (*TokenStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open token db: %w", err)
	}

	ts := &TokenStore{db: db}
	if err := ts.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate token db: %w", err)
	}
	return ts, nil
}

func (ts *TokenStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		email TEXT PRIMARY KEY,
		refresh_token TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS access_tokens (
		email TEXT PRIMARY KEY,
		access_token TEXT NOT NULL,
		token_type TEXT NOT NULL,
		expiry DATETIME NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := ts.db.Exec(schema)
	return err
}

// SaveRefreshToken upserts a refresh token for the given email.
func (ts *TokenStore) SaveRefreshToken(email, refreshToken string) error {
	_, err := ts.db.Exec(`
		INSERT INTO refresh_tokens (email, refresh_token, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(email) DO UPDATE SET
			refresh_token = excluded.refresh_token,
			updated_at = CURRENT_TIMESTAMP
	`, email, refreshToken)
	return err
}

// LoadRefreshToken retrieves the refresh token for the given email.
func (ts *TokenStore) LoadRefreshToken(email string) (string, error) {
	var refreshToken string
	err := ts.db.QueryRow(`SELECT refresh_token FROM refresh_tokens WHERE email = ?`, email).Scan(&refreshToken)
	if err != nil {
		return "", err
	}
	return refreshToken, nil
}

// SaveAccessToken upserts an access token for the given email.
func (ts *TokenStore) SaveAccessToken(email string, tok *oauth2.Token) error {
	_, err := ts.db.Exec(`
		INSERT INTO access_tokens (email, access_token, token_type, expiry, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(email) DO UPDATE SET
			access_token = excluded.access_token,
			token_type = excluded.token_type,
			expiry = excluded.expiry,
			updated_at = CURRENT_TIMESTAMP
	`, email, tok.AccessToken, tok.TokenType, tok.Expiry.UTC())
	return err
}

// LoadAccessToken retrieves the access token for the given email.
func (ts *TokenStore) LoadAccessToken(email string) (*oauth2.Token, error) {
	var accessToken, tokenType string
	var expiry time.Time
	err := ts.db.QueryRow(`SELECT access_token, token_type, expiry FROM access_tokens WHERE email = ?`, email).
		Scan(&accessToken, &tokenType, &expiry)
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken: accessToken,
		TokenType:   tokenType,
		Expiry:      expiry,
	}, nil
}

// LoadFullToken combines refresh and access tokens into a single oauth2.Token.
func (ts *TokenStore) LoadFullToken(email string) (*oauth2.Token, error) {
	refreshToken, err := ts.LoadRefreshToken(email)
	if err != nil {
		return nil, fmt.Errorf("load refresh token: %w", err)
	}

	tok, err := ts.LoadAccessToken(email)
	if err == sql.ErrNoRows {
		// Have refresh token but no access token yet — return expired token to force refresh.
		return &oauth2.Token{
			RefreshToken: refreshToken,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load access token: %w", err)
	}

	tok.RefreshToken = refreshToken
	return tok, nil
}

// Close closes the database connection.
func (ts *TokenStore) Close() error {
	return ts.db.Close()
}

// dbTokenSource wraps an oauth2.TokenSource and persists refreshed tokens back to the database.
type dbTokenSource struct {
	email    string
	store    *TokenStore
	base     oauth2.TokenSource
	mu       sync.Mutex
	current  *oauth2.Token
}

func newDBTokenSource(email string, store *TokenStore, base oauth2.TokenSource, initial *oauth2.Token) oauth2.TokenSource {
	return &dbTokenSource{
		email:   email,
		store:   store,
		base:    base,
		current: initial,
	}
}

func (s *dbTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If current token is still valid, return it.
	if s.current.Valid() {
		return s.current, nil
	}

	// Get a new token (triggers refresh via the base source).
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}

	// Persist the refreshed access token.
	if err := s.store.SaveAccessToken(s.email, tok); err != nil {
		return nil, fmt.Errorf("save refreshed access token: %w", err)
	}

	// If the refresh token was rotated, persist that too.
	if tok.RefreshToken != "" && tok.RefreshToken != s.current.RefreshToken {
		if err := s.store.SaveRefreshToken(s.email, tok.RefreshToken); err != nil {
			return nil, fmt.Errorf("save rotated refresh token: %w", err)
		}
	}

	s.current = tok
	return tok, nil
}
