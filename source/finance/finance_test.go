package finance

import (
	"context"
	"testing"
)

func TestName(t *testing.T) {
	f := New(Config{})
	if got := f.Name(); got != "finance" {
		t.Errorf("Name() = %q, want %q", got, "finance")
	}
}

func TestStatusAlwaysConnected(t *testing.T) {
	f := New(Config{})
	st, err := f.Status(context.Background(), nil)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.Connected {
		t.Error("Status.Connected = false, want true")
	}
}
