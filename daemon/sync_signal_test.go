package daemon

import "testing"

func TestSyncNotifier_SendReceive(t *testing.T) {
	n := NewSyncNotifier()
	n.Notify("gmail")
	sig := <-n.C()
	if sig.Source != "gmail" {
		t.Errorf("got source %q, want gmail", sig.Source)
	}
}

func TestSyncNotifier_NonBlockingWhenFull(t *testing.T) {
	n := NewSyncNotifier()
	// Fill the buffer (cap 16).
	for i := 0; i < 20; i++ {
		n.Notify("test")
	}
	// Should not panic or block.
	if len(n.ch) != 16 {
		t.Errorf("expected buffer full at 16, got %d", len(n.ch))
	}
}
