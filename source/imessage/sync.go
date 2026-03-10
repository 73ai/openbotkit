package imessage

import (
	"fmt"
	"log/slog"

	"github.com/priyanshujain/openbotkit/store"
)

func Sync(db *store.DB, opts SyncOptions) (*SyncResult, error) {
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	chatDB, err := OpenChatDB()
	if err != nil {
		return nil, fmt.Errorf("open chat.db: %w", err)
	}
	defer chatDB.Close()

	slog.Info("imessage: fetching handles")
	handles, err := FetchHandles(chatDB)
	if err != nil {
		return nil, fmt.Errorf("fetch handles: %w", err)
	}
	slog.Info("imessage: found handles", "count", len(handles))

	for i := range handles {
		if err := SaveHandle(db, &handles[i]); err != nil {
			slog.Error("imessage: save handle", "id", handles[i].ID, "error", err)
		}
	}

	slog.Info("imessage: fetching chats")
	chats, err := FetchChats(chatDB)
	if err != nil {
		return nil, fmt.Errorf("fetch chats: %w", err)
	}
	slog.Info("imessage: found chats", "count", len(chats))

	for i := range chats {
		if err := SaveChat(db, &chats[i]); err != nil {
			slog.Error("imessage: save chat", "guid", chats[i].GUID, "error", err)
		}
	}

	sinceROWID := int64(0)
	if !opts.Full {
		sinceROWID, _ = MaxAppleROWID(db)
	}
	slog.Info("imessage: fetching messages", "since_rowid", sinceROWID)

	messages, skipped, err := FetchMessagesSince(chatDB, sinceROWID)
	if err != nil {
		return nil, fmt.Errorf("fetch messages: %w", err)
	}
	slog.Info("imessage: found messages", "count", len(messages), "skipped_null_text", skipped)

	result := &SyncResult{Skipped: skipped}
	for i := range messages {
		if err := SaveMessage(db, &messages[i]); err != nil {
			slog.Error("imessage: save message", "guid", messages[i].GUID, "error", err)
			result.Errors++
			continue
		}
		result.Synced++
	}

	return result, nil
}
