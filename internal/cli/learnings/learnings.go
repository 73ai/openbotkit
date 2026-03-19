package learnings

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/73ai/openbotkit/config"
	learningssvc "github.com/73ai/openbotkit/service/learnings"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "learnings",
	Short: "Manage saved learnings",
}

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the learnings directory in your file browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := config.LearningsDir()
		var opener string
		switch runtime.GOOS {
		case "darwin":
			opener = "open"
		default:
			opener = "xdg-open"
		}
		return exec.Command(opener, dir).Start()
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all learning topics",
	RunE: func(cmd *cobra.Command, args []string) error {
		st := learningssvc.New(config.LearningsDir())
		topics, err := st.List()
		if err != nil {
			return err
		}
		if len(topics) == 0 {
			fmt.Println("No learnings saved yet.")
			return nil
		}
		for _, t := range topics {
			fmt.Println(t)
		}
		return nil
	},
}

func init() {
	Cmd.AddCommand(openCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(searchCmd)
}
