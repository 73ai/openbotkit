package tty

import (
	"fmt"
	"os"
)

// IsInteractive returns true if both stdin and stdout are terminals.
func IsInteractive() bool {
	return isTerminal(os.Stdin.Fd()) && isTerminal(os.Stdout.Fd())
}

// RequireInteractive returns an error if the session is not interactive,
// with a hint about the non-interactive equivalent command.
func RequireInteractive(hint string) error {
	if IsInteractive() {
		return nil
	}
	return fmt.Errorf("this command requires an interactive terminal. Use: %s", hint)
}
