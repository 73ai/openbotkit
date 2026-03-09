package memory

import "time"

type Category string

const (
	CategoryIdentity     Category = "identity"
	CategoryPreference   Category = "preference"
	CategoryRelationship Category = "relationship"
	CategoryProject      Category = "project"
)

type Memory struct {
	ID        int64
	Content   string
	Category  Category
	Source    string // "history", "whatsapp", "gmail", "applenotes", "manual"
	SourceRef string // optional reference (session_id, etc.)
	CreatedAt time.Time
	UpdatedAt time.Time
}
