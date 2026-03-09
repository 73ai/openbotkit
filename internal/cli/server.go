package cli

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/internal/platform"
	"github.com/priyanshujain/openbotkit/internal/server"
)

var serverAddr string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the obk HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := cfg.RequireSetup(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), platform.ShutdownSignals...)
		defer stop()

		s := server.New(cfg, serverAddr)
		return s.Run(ctx)
	},
}

func init() {
	serverCmd.Flags().StringVar(&serverAddr, "addr", ":8443", "address to listen on")
	rootCmd.AddCommand(serverCmd)
}
