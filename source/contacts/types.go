package contacts

import "time"

type Contact struct {
	ID           int64         `json:"id"`
	DisplayName  string        `json:"display_name"`
	Identities   []Identity    `json:"identities,omitempty"`
	Aliases      []string      `json:"aliases,omitempty"`
	Interactions []Interaction `json:"interactions,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type Identity struct {
	ID            int64  `json:"id"`
	ContactID     int64  `json:"contact_id"`
	Source        string `json:"source"`
	IdentityType  string `json:"identity_type"`
	IdentityValue string `json:"identity_value"`
	DisplayName   string `json:"display_name,omitempty"`
	RawValue      string `json:"raw_value,omitempty"`
}

type Interaction struct {
	Channel           string     `json:"channel"`
	MessageCount      int        `json:"message_count"`
	LastInteractionAt *time.Time `json:"last_interaction_at,omitempty"`
}

type SearchResult struct {
	Contact      Contact `json:"contact"`
	MatchScore   int     `json:"match_score"`
	MatchedAlias string  `json:"matched_alias"`
}

type SyncResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Linked  int `json:"linked"`
	Errors  int `json:"errors"`
}

type SyncOptions struct {
	Sources []string // empty = all available
}
