package tools

import (
	"context"
	"errors"
	"testing"
)

type mockInteractor struct {
	notified   []string
	links      []struct{ text, url string }
	approvals  []string
	approveAll bool
	approveErr error
}

func (m *mockInteractor) Notify(msg string) error {
	m.notified = append(m.notified, msg)
	return nil
}

func (m *mockInteractor) NotifyLink(text, url string) error {
	m.links = append(m.links, struct{ text, url string }{text, url})
	return nil
}

func (m *mockInteractor) RequestApproval(desc string) (bool, error) {
	m.approvals = append(m.approvals, desc)
	if m.approveErr != nil {
		return false, m.approveErr
	}
	return m.approveAll, nil
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
