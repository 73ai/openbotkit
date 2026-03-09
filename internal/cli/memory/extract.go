package memory

import (
	"context"
	"fmt"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/provider"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
	"github.com/priyanshujain/openbotkit/store"
	"github.com/spf13/cobra"

	_ "github.com/priyanshujain/openbotkit/provider/anthropic"
	_ "github.com/priyanshujain/openbotkit/provider/gemini"
	_ "github.com/priyanshujain/openbotkit/provider/openai"
)

var extractLast int

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract personal facts from conversation history",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := cfg.RequireSetup(); err != nil {
			return err
		}

		// Open history DB.
		histDB, err := store.Open(store.Config{
			Driver: cfg.History.Storage.Driver,
			DSN:    cfg.HistoryDataDSN(),
		})
		if err != nil {
			return fmt.Errorf("open history database: %w", err)
		}
		defer histDB.Close()

		// Load recent messages from history.
		messages, err := loadRecentMessages(histDB, extractLast)
		if err != nil {
			return fmt.Errorf("load messages: %w", err)
		}

		if len(messages) == 0 {
			fmt.Println("No messages found in history.")
			return nil
		}

		// Open user_memory DB.
		if err := config.EnsureSourceDir("user_memory"); err != nil {
			return fmt.Errorf("ensure user_memory dir: %w", err)
		}

		memDB, err := store.Open(store.Config{
			Driver: cfg.UserMemory.Storage.Driver,
			DSN:    cfg.UserMemoryDataDSN(),
		})
		if err != nil {
			return fmt.Errorf("open memory database: %w", err)
		}
		defer memDB.Close()

		if err := memory.Migrate(memDB); err != nil {
			return fmt.Errorf("migrate memory: %w", err)
		}

		// Create LLM client.
		llm, err := buildLLM(cfg)
		if err != nil {
			return fmt.Errorf("build LLM: %w", err)
		}

		ctx := context.Background()

		// Extract candidate facts.
		candidates, err := memory.Extract(ctx, llm, messages)
		if err != nil {
			return fmt.Errorf("extract: %w", err)
		}

		if len(candidates) == 0 {
			fmt.Println("No personal facts found.")
			return nil
		}

		// Reconcile with existing memories.
		result, err := memory.Reconcile(ctx, memDB, llm, candidates)
		if err != nil {
			return fmt.Errorf("reconcile: %w", err)
		}

		fmt.Printf("Added %d, Updated %d, Deleted %d, Skipped %d\n",
			result.Added, result.Updated, result.Deleted, result.Skipped)
		return nil
	},
}

func loadRecentMessages(db *store.DB, lastN int) ([]string, error) {
	if err := historysrc.Migrate(db); err != nil {
		return nil, fmt.Errorf("migrate history: %w", err)
	}

	query := db.Rebind(`
		SELECT m.content FROM history_messages m
		JOIN history_conversations c ON c.id = m.conversation_id
		WHERE m.role = 'user'
		ORDER BY c.updated_at DESC, m.timestamp DESC
		LIMIT ?`)

	rows, err := db.Query(query, lastN*50)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, content)
	}
	return messages, rows.Err()
}

func buildLLM(cfg *config.Config) (memory.LLM, error) {
	registry, err := provider.NewRegistry(cfg.Models)
	if err != nil {
		return nil, fmt.Errorf("create provider registry: %w", err)
	}
	router := provider.NewRouter(registry, cfg.Models)
	return &memory.RouterLLM{Router: router, Tier: provider.TierFast}, nil
}

func init() {
	Cmd.AddCommand(extractCmd)
	extractCmd.Flags().IntVar(&extractLast, "last", 1, "number of recent sessions to extract from")
}
