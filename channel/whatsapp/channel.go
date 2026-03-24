package whatsapp

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// messageSender abstracts WhatsApp message sending for testing.
type messageSender interface {
	SendText(ctx context.Context, jid, text string) error
}

type incomingMessage struct {
	text      string
	senderJID string
	messageID string
}

// Channel implements channel.Channel for WhatsApp.
type Channel struct {
	sender   messageSender
	ownerJID string
	incoming chan incomingMessage
	done     chan struct{}

	approvalMu sync.Mutex
	approvalCh chan bool
}

func NewChannel(sender messageSender, ownerJID string) *Channel {
	return &Channel{
		sender:   sender,
		ownerJID: ownerJID,
		incoming: make(chan incomingMessage, 16),
		done:     make(chan struct{}),
	}
}

func (c *Channel) Send(msg string) error {
	return c.sender.SendText(context.Background(), c.ownerJID, msg)
}

func (c *Channel) Receive() (string, error) {
	msg, ok := <-c.incoming
	if !ok {
		return "", io.EOF
	}
	return msg.text, nil
}

func (c *Channel) RequestApproval(action string) (bool, error) {
	c.approvalMu.Lock()
	c.approvalCh = make(chan bool, 1)
	c.approvalMu.Unlock()

	prompt := fmt.Sprintf("Approve action?\n\n%s\n\nReply 1 to approve, 2 to deny.", action)
	if err := c.sender.SendText(context.Background(), c.ownerJID, prompt); err != nil {
		return false, fmt.Errorf("send approval request: %w", err)
	}

	c.approvalMu.Lock()
	ch := c.approvalCh
	c.approvalMu.Unlock()

	approved := <-ch
	return approved, nil
}

func (c *Channel) SendLink(text string, url string) error {
	return c.sender.SendText(context.Background(), c.ownerJID, text+"\n"+url)
}

// HandleIncoming routes an incoming message. If an approval is pending and
// the text is "1" or "2", it routes to the approval handler. Otherwise the
// message is queued for Receive().
func (c *Channel) HandleIncoming(text, senderJID, messageID string) {
	c.approvalMu.Lock()
	ch := c.approvalCh
	c.approvalMu.Unlock()

	if ch != nil {
		switch text {
		case "1":
			c.approvalMu.Lock()
			c.approvalCh = nil
			c.approvalMu.Unlock()
			ch <- true
			return
		case "2":
			c.approvalMu.Lock()
			c.approvalCh = nil
			c.approvalMu.Unlock()
			ch <- false
			return
		}
	}

	c.incoming <- incomingMessage{text: text, senderJID: senderJID, messageID: messageID}
}

// CancelPendingApproval denies any pending approval (used on kill).
func (c *Channel) CancelPendingApproval() {
	c.approvalMu.Lock()
	ch := c.approvalCh
	c.approvalCh = nil
	c.approvalMu.Unlock()
	if ch != nil {
		ch <- false
	}
}

// Close shuts down the incoming channel.
func (c *Channel) Close() {
	close(c.incoming)
}
