package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

type mockInteractor struct {
	mu         sync.Mutex
	notified   []string
	links      []struct{ text, url string }
	approvals  []string
	approveAll bool
	approveErr error
	notifyErr  error
	linkCh     chan struct{ text, url string } // optional signal for NotifyLink
}

func (m *mockInteractor) Notify(msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notified = append(m.notified, msg)
	return m.notifyErr
}

func (m *mockInteractor) NotifyLink(text, url string) error {
	m.mu.Lock()
	m.links = append(m.links, struct{ text, url string }{text, url})
	ch := m.linkCh
	m.mu.Unlock()
	if ch != nil {
		ch <- struct{ text, url string }{text, url}
	}
	return nil
}

func (m *mockInteractor) RequestApproval(desc string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.approvals = append(m.approvals, desc)
	if m.approveErr != nil {
		return false, m.approveErr
	}
	return m.approveAll, nil
}

func TestGuardedAction_LowRisk_AutoApproves(t *testing.T) {
	inter := &mockInteractor{}
	ran := false
	result, err := GuardedAction(context.Background(), inter, RiskLow, "react :thumbsup:", func() (string, error) {
		ran = true
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("GuardedAction: %v", err)
	}
	if !ran {
		t.Error("action was not executed")
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if len(inter.approvals) != 0 {
		t.Error("low risk should not request approval")
	}
	if len(inter.notified) != 1 || inter.notified[0] != "react :thumbsup:" {
		t.Errorf("notified = %v, want description notification", inter.notified)
	}
}

func TestGuardedAction_HighRisk_RequestsApproval(t *testing.T) {
	inter := &mockInteractor{approveAll: true}
	ran := false
	result, err := GuardedAction(context.Background(), inter, RiskHigh, "send email", func() (string, error) {
		ran = true
		return "sent", nil
	})
	if err != nil {
		t.Fatalf("GuardedAction: %v", err)
	}
	if !ran {
		t.Error("action was not executed")
	}
	if result != "sent" {
		t.Errorf("result = %q, want %q", result, "sent")
	}
	if len(inter.approvals) != 1 {
		t.Errorf("expected 1 approval request, got %d", len(inter.approvals))
	}
}

func TestGuardedAction_HighRisk_Denied(t *testing.T) {
	inter := &mockInteractor{approveAll: false}
	ran := false
	result, err := GuardedAction(context.Background(), inter, RiskHigh, "delete all", func() (string, error) {
		ran = true
		return "", nil
	})
	if err != nil {
		t.Fatalf("GuardedAction: %v", err)
	}
	if ran {
		t.Error("action should not run when denied")
	}
	if result != "denied_by_user" {
		t.Errorf("result = %q, want %q", result, "denied_by_user")
	}
}

func TestGuardedAction_AutoApproveByRules(t *testing.T) {
	rules := NewApprovalRuleSet()
	rules.Add(ApprovalRule{ToolName: "slack_send", Pattern: "#general"})
	inter := &mockInteractor{approveAll: false}
	input, _ := json.Marshal(map[string]string{"channel": "#general"})

	ran := false
	result, err := GuardedAction(context.Background(), inter, RiskMedium, "send message",
		func() (string, error) {
			ran = true
			return "sent", nil
		},
		WithApprovalRules(rules, "slack_send", input),
	)
	if err != nil {
		t.Fatalf("GuardedAction: %v", err)
	}
	if !ran {
		t.Error("action should have been auto-approved by rules")
	}
	if result != "sent" {
		t.Errorf("result = %q, want %q", result, "sent")
	}
	if len(inter.approvals) != 0 {
		t.Error("should not have requested approval when rules match")
	}
	if len(inter.notified) < 1 || !strings.Contains(inter.notified[0], "Auto-approved") {
		t.Errorf("expected auto-approve notification, got %v", inter.notified)
	}
}

func TestGuardedAction_RubberStampWarning(t *testing.T) {
	rules := NewApprovalRuleSet()
	inter := &mockInteractor{approveAll: true}

	// Use different channels each time to avoid triggering auto-approve rules.
	channels := []string{"#ch1", "#ch2", "#ch3", "#ch4", "#ch5", "#ch6"}
	for _, ch := range channels {
		input, _ := json.Marshal(map[string]string{"channel": ch})
		_, _ = GuardedAction(context.Background(), inter, RiskMedium, "send to "+ch,
			func() (string, error) { return "ok", nil },
			WithApprovalRules(rules, "slack_send", input),
		)
	}

	foundWarning := false
	for _, n := range inter.notified {
		if strings.Contains(n, "approved several actions quickly") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Error("expected rubber-stamping warning")
	}
}

func TestGuardedWrite_Approved(t *testing.T) {
	inter := &mockInteractor{approveAll: true}
	ran := false
	result, err := GuardedWrite(context.Background(), inter, "send email", func() (string, error) {
		ran = true
		return "sent", nil
	})
	if err != nil {
		t.Fatalf("GuardedWrite: %v", err)
	}
	if !ran {
		t.Error("action was not executed")
	}
	if result != "sent" {
		t.Errorf("result = %q, want %q", result, "sent")
	}
	if len(inter.notified) != 1 || inter.notified[0] != "Done." {
		t.Errorf("notified = %v", inter.notified)
	}
}

func TestGuardedWrite_Denied(t *testing.T) {
	inter := &mockInteractor{approveAll: false}
	ran := false
	result, err := GuardedWrite(context.Background(), inter, "delete file", func() (string, error) {
		ran = true
		return "done", nil
	})
	if err != nil {
		t.Fatalf("GuardedWrite: %v", err)
	}
	if ran {
		t.Error("action should not have been executed")
	}
	if result != "denied_by_user" {
		t.Errorf("result = %q, want %q", result, "denied_by_user")
	}
	if len(inter.notified) != 1 || inter.notified[0] != "Action not performed." {
		t.Errorf("notified = %v", inter.notified)
	}
}

func TestGuardedWrite_ApprovalError(t *testing.T) {
	want := errors.New("connection lost")
	inter := &mockInteractor{approveErr: want}
	_, err := GuardedWrite(context.Background(), inter, "action", func() (string, error) {
		return "", nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, want) {
		t.Errorf("error = %v, want %v", err, want)
	}
}

func TestGuardedWrite_NotifyError(t *testing.T) {
	want := errors.New("channel closed")
	inter := &mockInteractor{approveAll: true, notifyErr: want}
	_, err := GuardedWrite(context.Background(), inter, "action", func() (string, error) {
		return "ok", nil
	})
	if err == nil {
		t.Fatal("expected error from Notify")
	}
	if !errors.Is(err, want) {
		t.Errorf("error = %v, want %v", err, want)
	}
}
