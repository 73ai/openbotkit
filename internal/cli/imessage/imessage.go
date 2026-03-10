package imessage

import (
	"fmt"

	"github.com/priyanshujain/openbotkit/config"
	imsrc "github.com/priyanshujain/openbotkit/source/imessage"
	"github.com/priyanshujain/openbotkit/store"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "imessage",
	Short: "Manage iMessage data source",
}

func init() {
	Cmd.AddCommand(syncCmd)
	Cmd.AddCommand(messagesCmd)
	Cmd.AddCommand(chatsCmd)
}

func openIMessageDB(cfg *config.Config) (*store.DB, error) {
	if err := config.EnsureSourceDir("imessage"); err != nil {
		return nil, fmt.Errorf("create imessage dir: %w", err)
	}

	dsn := cfg.IMessageDataDSN()
	db, err := store.Open(store.Config{
		Driver: cfg.IMessage.Storage.Driver,
		DSN:    dsn,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := imsrc.Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return db, nil
}
