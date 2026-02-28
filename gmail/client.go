package gmail

import (
	"context"
	"fmt"

	"google.golang.org/api/gmail/v1"
)

// Account represents an authenticated Gmail account.
type Account struct {
	Email   string
	Service *gmail.Service
}

// Client holds authenticated connections to one or more Gmail accounts.
type Client struct {
	Accounts []*Account
}

// NewClient authenticates each configured account and returns a Client.
// credentialsFile is the path to the Google OAuth client credentials JSON.
// credDBPath is the path to the SQLite database for storing tokens.
// emails is the list of email addresses to authenticate.
func NewClient(credentialsFile string, credDBPath string, emails []string) (*Client, error) {
	tokenStore, err := NewTokenStore(credDBPath)
	if err != nil {
		return nil, fmt.Errorf("open token store: %w", err)
	}
	defer tokenStore.Close()

	ctx := context.Background()
	c := &Client{}

	for _, email := range emails {
		fmt.Printf("Authenticating %s...\n", email)
		srv, err := authenticate(ctx, credentialsFile, email, tokenStore)
		if err != nil {
			return nil, fmt.Errorf("authenticate %s: %w", email, err)
		}
		c.Accounts = append(c.Accounts, &Account{
			Email:   email,
			Service: srv,
		})
	}

	return c, nil
}
