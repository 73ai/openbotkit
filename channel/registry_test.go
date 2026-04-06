package channel

import (
	"context"
	"testing"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	p := &regTestPusher{}
	reg.Register("telegram", p)

	got, err := reg.Get("telegram")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != p {
		t.Error("returned pusher does not match registered one")
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Get("slack")
	if err == nil {
		t.Error("expected error for unregistered channel")
	}
}

type regTestPusher struct{}

func (r *regTestPusher) Push(_ context.Context, _ string) error { return nil }
