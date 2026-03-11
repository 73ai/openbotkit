package tools

import "github.com/priyanshujain/openbotkit/oauth/google"

// ScopeChecker abstracts scope queries for testability.
type ScopeChecker interface {
	HasScopes(account string, required []string) (bool, error)
}

// GoogleScopeChecker wraps a Google token store for scope checking.
type GoogleScopeChecker struct {
	TokenDBPath string
}

func (c *GoogleScopeChecker) HasScopes(account string, required []string) (bool, error) {
	store, err := google.NewTokenStore(c.TokenDBPath)
	if err != nil {
		return false, err
	}
	defer store.Close()
	return store.HasScopes(account, required)
}
