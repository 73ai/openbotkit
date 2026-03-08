package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestSend(t *testing.T) {
	var buf bytes.Buffer
	ch := New(strings.NewReader(""), &buf)

	if err := ch.Send("Hello!"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := buf.String(); got != "Hello!\n" {
		t.Errorf("output = %q", got)
	}
}

func TestReceive(t *testing.T) {
	var buf bytes.Buffer
	ch := New(strings.NewReader("hello\n"), &buf)

	msg, err := ch.Receive()
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if msg != "hello" {
		t.Errorf("msg = %q", msg)
	}
}

func TestReceiveEOF(t *testing.T) {
	var buf bytes.Buffer
	ch := New(strings.NewReader(""), &buf)

	_, err := ch.Receive()
	if err != io.EOF {
		t.Errorf("err = %v, want io.EOF", err)
	}
}

func TestRequestApproval_Yes(t *testing.T) {
	var buf bytes.Buffer
	ch := New(strings.NewReader("y\n"), &buf)

	approved, err := ch.RequestApproval("run bash command")
	if err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}
	if !approved {
		t.Error("expected approval")
	}
}

func TestRequestApproval_No(t *testing.T) {
	var buf bytes.Buffer
	ch := New(strings.NewReader("n\n"), &buf)

	approved, err := ch.RequestApproval("run bash command")
	if err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}
	if approved {
		t.Error("expected denial")
	}
}
