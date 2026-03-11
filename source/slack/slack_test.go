package slack

import (
	"context"
	"testing"
)

func TestSlack_Name(t *testing.T) {
	s := New(Config{})
	if s.Name() != "slack" {
		t.Errorf("Name() = %q", s.Name())
	}
}

func TestSlack_Status_NilConfig(t *testing.T) {
	s := New(Config{})
	st, err := s.Status(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if st.Connected {
		t.Error("should not be connected with nil config")
	}
}
