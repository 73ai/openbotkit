package channel

// Channel defines how the agent communicates with the user.
type Channel interface {
	// Send displays a message to the user.
	Send(msg string) error

	// Receive waits for and returns the next user input.
	// Returns empty string and io.EOF on end of input.
	Receive() (string, error)

	// RequestApproval asks the user to approve an action.
	// Returns true if approved, false if denied.
	RequestApproval(action string) (bool, error)

	// SendLink sends a message with a clickable link.
	SendLink(text string, url string) error
}
