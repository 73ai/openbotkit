package cli

import (
	"fmt"
	"os"

	applenotescli "github.com/priyanshujain/openbotkit/internal/cli/applenotes"
	authcli "github.com/priyanshujain/openbotkit/internal/cli/auth"
	financecli "github.com/priyanshujain/openbotkit/internal/cli/finance"
	"github.com/priyanshujain/openbotkit/internal/cli/gmail"
	historycli "github.com/priyanshujain/openbotkit/internal/cli/history"
	imessagecli "github.com/priyanshujain/openbotkit/internal/cli/imessage"
	memorycli "github.com/priyanshujain/openbotkit/internal/cli/memory"
	websearchcli "github.com/priyanshujain/openbotkit/internal/cli/websearch"
	whatsappcli "github.com/priyanshujain/openbotkit/internal/cli/whatsapp"
	"github.com/spf13/cobra"
)

var Version = "dev" // set via -ldflags

var rootCmd = &cobra.Command{
	Use:   "obk",
	Short: "OpenBotKit — toolkit for building AI personal assistants",
	Long:  "OpenBotKit (obk) is a toolkit for building AI personal assistants through data source integrations.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the obk version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("obk version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(applenotescli.Cmd)
	rootCmd.AddCommand(authcli.Cmd)
	rootCmd.AddCommand(financecli.Cmd)
	rootCmd.AddCommand(imessagecli.Cmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(gmail.Cmd)
	rootCmd.AddCommand(historycli.Cmd)
	rootCmd.AddCommand(memorycli.Cmd)
	rootCmd.AddCommand(websearchcli.Cmd)
	rootCmd.AddCommand(whatsappcli.Cmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
