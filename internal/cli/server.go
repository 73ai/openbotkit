package cli

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/daemon/service"
	"github.com/priyanshujain/openbotkit/internal/platform"
	"github.com/priyanshujain/openbotkit/internal/server"
)

var serverAddr string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage the obk HTTP API server",
}

var serverRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the server process (used internally by the system service)",
	Hidden: true,
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

var serverInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the server as a system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager("server")
		if err != nil {
			return err
		}

		cfg, err := service.DefaultConfig("server", []string{"server", "run"})
		if err != nil {
			return err
		}

		if err := mgr.Install(cfg); err != nil {
			return fmt.Errorf("install server: %w", err)
		}

		fmt.Printf("server installed (platform: %s)\n", service.DetectPlatform())
		return nil
	},
}

var serverUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the server system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager("server")
		if err != nil {
			return err
		}

		if err := mgr.Uninstall(); err != nil {
			return fmt.Errorf("uninstall server: %w", err)
		}

		fmt.Println("server uninstalled")
		return nil
	},
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager("server")
		if err != nil {
			return err
		}

		if err := mgr.Start(); err != nil {
			return fmt.Errorf("start server: %w", err)
		}

		fmt.Println("server started")
		return nil
	},
}

var serverStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the server service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager("server")
		if err != nil {
			return err
		}

		if err := mgr.Stop(); err != nil {
			return fmt.Errorf("stop server: %w", err)
		}

		fmt.Println("server stopped")
		return nil
	},
}

var serverRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the server service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager("server")
		if err != nil {
			return err
		}

		if err := mgr.Stop(); err != nil {
			return fmt.Errorf("stop server: %w", err)
		}

		if err := mgr.Start(); err != nil {
			return fmt.Errorf("start server: %w", err)
		}

		fmt.Println("server restarted")
		return nil
	},
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the server service status",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager("server")
		if err != nil {
			return err
		}

		status, err := mgr.Status()
		if err != nil {
			return fmt.Errorf("check status: %w", err)
		}

		fmt.Printf("server: %s\n", status)
		return nil
	},
}

func init() {
	serverRunCmd.Flags().StringVar(&serverAddr, "addr", ":8443", "address to listen on")
	serverCmd.AddCommand(serverRunCmd)
	serverCmd.AddCommand(serverInstallCmd)
	serverCmd.AddCommand(serverUninstallCmd)
	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStopCmd)
	serverCmd.AddCommand(serverRestartCmd)
	serverCmd.AddCommand(serverStatusCmd)
	serverCmd.AddCommand(newLogsCmd("server"))
	rootCmd.AddCommand(serverCmd)
}
