package applenotes

import "time"

type Config struct{}

type Note struct {
	AppleID           string
	Title             string
	Body              string
	Folder            string
	FolderID          string
	Account           string
	PasswordProtected bool
	CreatedAt         time.Time
	ModifiedAt        time.Time
}

type Folder struct {
	AppleID       string
	Name          string
	ParentAppleID string
	Account       string
}

type SyncOptions struct {
	Full bool
}

type SyncResult struct {
	Synced  int
	Skipped int
	Errors  int
}

type ListOptions struct {
	Folder string
	Limit  int
	Offset int
}
