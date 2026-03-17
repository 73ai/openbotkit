package spectest

import (
	"testing"

	"github.com/73ai/openbotkit/agent"
)

type Email struct {
	MessageID string
	Account   string
	From      string
	To        string
	Subject   string
	Body      string
}

type WhatsAppMessage struct {
	MessageID  string
	ChatJID    string
	ChatName   string
	SenderJID  string
	SenderName string
	Text       string
	IsFromMe   bool
}

type UserMemory struct {
	Content  string
	Category string // identity, preference, relationship, project
}

type ContactFixture struct {
	Name        string
	Phones      []string
	Emails      []string
	WhatsAppJID string
	Aliases     []string
}

type Fixture interface {
	Agent(t *testing.T) *agent.Agent
	GivenEmails(t *testing.T, emails []Email)
	GivenWhatsAppMessages(t *testing.T, messages []WhatsAppMessage)
	GivenMemories(t *testing.T, memories []UserMemory)
	GivenContacts(t *testing.T, contacts []ContactFixture)
}
