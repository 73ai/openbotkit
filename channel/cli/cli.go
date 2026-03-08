package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Channel implements channel.Channel for terminal stdin/stdout.
type Channel struct {
	reader *bufio.Reader
	writer io.Writer
}

// New creates a CLI channel that reads from r and writes to w.
func New(r io.Reader, w io.Writer) *Channel {
	return &Channel{
		reader: bufio.NewReader(r),
		writer: w,
	}
}

// Send writes a message to the output.
func (c *Channel) Send(msg string) error {
	_, err := fmt.Fprintln(c.writer, msg)
	return err
}

// Receive reads a line of input from the user.
func (c *Channel) Receive() (string, error) {
	fmt.Fprint(c.writer, "> ")
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// RequestApproval prompts the user for yes/no confirmation.
func (c *Channel) RequestApproval(action string) (bool, error) {
	fmt.Fprintf(c.writer, "Approve: %s [y/n] ", action)
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes", nil
}
