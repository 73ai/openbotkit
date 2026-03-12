package tools

import "testing"

func TestAutoApproveInteractor(t *testing.T) {
	var i AutoApproveInteractor

	approved, err := i.RequestApproval("dangerous action")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Fatal("expected approval")
	}

	if err := i.Notify("msg"); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if err := i.NotifyLink("text", "http://example.com"); err != nil {
		t.Fatalf("notify link: %v", err)
	}
}
