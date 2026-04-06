package daemon

import "testing"

func TestSyncNotifier_SendReceive(t *testing.T) {
	n := NewSyncNotifier()
	ch := n.Subscribe()
	n.Notify("gmail")
	sig := <-ch
	if sig.Source != "gmail" {
		t.Errorf("got source %q, want gmail", sig.Source)
	}
}

func TestSyncNotifier_NonBlockingWhenFull(t *testing.T) {
	n := NewSyncNotifier()
	ch := n.Subscribe()
	// Fill the buffer (cap 16).
	for i := 0; i < 20; i++ {
		n.Notify("test")
	}
	// Should not panic or block.
	if len(ch) != 16 {
		t.Errorf("expected buffer full at 16, got %d", len(ch))
	}
}

func TestSyncNotifier_FanOut(t *testing.T) {
	n := NewSyncNotifier()
	ch1 := n.Subscribe()
	ch2 := n.Subscribe()
	n.Notify("gmail")
	if sig := <-ch1; sig.Source != "gmail" {
		t.Errorf("ch1: got %q, want gmail", sig.Source)
	}
	if sig := <-ch2; sig.Source != "gmail" {
		t.Errorf("ch2: got %q, want gmail", sig.Source)
	}
}

func TestSyncNotifier_WithData(t *testing.T) {
	n := NewSyncNotifier()
	ch := n.Subscribe()
	ids := []int64{1, 2, 3}
	n.NotifyWithData("gmail", ids)
	sig := <-ch
	if sig.Source != "gmail" {
		t.Errorf("got source %q, want gmail", sig.Source)
	}
	got, ok := sig.Data.([]int64)
	if !ok || len(got) != 3 {
		t.Errorf("expected []int64 with 3 items, got %v", sig.Data)
	}
}
