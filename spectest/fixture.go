package spectest

import (
	"testing"

	"github.com/priyanshujain/openbotkit/agent"
)

type Email struct {
	MessageID string
	Account   string
	From      string
	To        string
	Subject   string
	Body      string
}

type Fixture interface {
	Agent(t *testing.T) *agent.Agent
	GivenEmails(t *testing.T, emails []Email)
}
