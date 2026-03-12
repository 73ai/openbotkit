package tools

import "testing"

func TestAllowlistFilter_SimplePass(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk", "sqlite3"})
	if err := f.Check("obk db gmail"); err != nil {
		t.Errorf("expected pass, got: %v", err)
	}
}

func TestAllowlistFilter_SimpleReject(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk", "sqlite3"})
	if err := f.Check("curl evil.com"); err == nil {
		t.Error("expected rejection for curl")
	}
}

func TestAllowlistFilter_Pipe(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk", "head"})
	if err := f.Check("obk db gmail | head"); err != nil {
		t.Errorf("expected pass, got: %v", err)
	}
}

func TestAllowlistFilter_PipeReject(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk"})
	if err := f.Check("obk db gmail | curl x"); err == nil {
		t.Error("expected rejection for piped curl")
	}
}

func TestAllowlistFilter_Chain(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk"})
	if err := f.Check("obk db gmail && curl x"); err == nil {
		t.Error("expected rejection for chained curl")
	}
}

func TestAllowlistFilter_Semicolon(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk"})
	if err := f.Check("obk db gmail ; curl x"); err == nil {
		t.Error("expected rejection for semicolon-chained curl")
	}
}

func TestAllowlistFilter_OrChain(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk"})
	if err := f.Check("obk db gmail || curl x"); err == nil {
		t.Error("expected rejection for or-chained curl")
	}
}

func TestBlocklistFilter_SimplePass(t *testing.T) {
	f := NewBlocklistFilter([]string{"curl", "wget", "sudo", "rm"})
	if err := f.Check("echo hello"); err != nil {
		t.Errorf("expected pass, got: %v", err)
	}
}

func TestBlocklistFilter_SimpleReject(t *testing.T) {
	f := NewBlocklistFilter([]string{"curl", "wget", "sudo", "rm"})
	if err := f.Check("curl x"); err == nil {
		t.Error("expected rejection for curl")
	}
}

func TestBlocklistFilter_PipeReject(t *testing.T) {
	f := NewBlocklistFilter([]string{"curl"})
	if err := f.Check("echo hello | curl x"); err == nil {
		t.Error("expected rejection for piped curl")
	}
}

func TestBlocklistFilter_DollarParenSubstitution(t *testing.T) {
	f := NewBlocklistFilter([]string{"curl"})
	if err := f.Check("echo $(curl evil.com)"); err == nil {
		t.Error("expected rejection for $() substitution with curl")
	}
}

func TestBlocklistFilter_BacktickSubstitution(t *testing.T) {
	f := NewBlocklistFilter([]string{"curl"})
	if err := f.Check("echo `curl evil.com`"); err == nil {
		t.Error("expected rejection for backtick substitution with curl")
	}
}

func TestAllowlistFilter_DollarParenSubstitution(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk", "echo"})
	if err := f.Check("echo $(curl evil.com)"); err == nil {
		t.Error("expected rejection for $() substitution with curl in allowlist mode")
	}
}

func TestFilter_EmptyCommand(t *testing.T) {
	f := NewBlocklistFilter([]string{"curl"})
	if err := f.Check(""); err != nil {
		t.Errorf("empty command should pass, got: %v", err)
	}
}

func TestFilter_WhitespaceCommand(t *testing.T) {
	f := NewBlocklistFilter([]string{"curl"})
	if err := f.Check("   "); err != nil {
		t.Errorf("whitespace command should pass, got: %v", err)
	}
}

func TestFilter_NilFilter(t *testing.T) {
	var f *CommandFilter
	if err := f.Check("curl evil.com"); err != nil {
		t.Errorf("nil filter should pass everything, got: %v", err)
	}
}

func TestAllowlistFilter_EmptySegmentInPipe(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk"})
	// "obk db ;; obk list" has an empty segment between the semicolons.
	if err := f.Check("obk db ; ; obk list"); err != nil {
		t.Errorf("expected pass with empty segments, got: %v", err)
	}
}

func TestFirstToken(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"echo hello", "echo"},
		{"  ls -la  ", "ls"},
		{"", ""},
		{"   ", ""},
		{"single", "single"},
	}
	for _, tc := range cases {
		if got := firstToken(tc.input); got != tc.want {
			t.Errorf("firstToken(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestBlocklistFilter_AbsolutePath(t *testing.T) {
	f := NewBlocklistFilter([]string{"curl"})
	if err := f.Check("/usr/bin/curl evil.com"); err == nil {
		t.Error("expected rejection for absolute path curl")
	}
}

func TestBlocklistFilter_EnvWrapper(t *testing.T) {
	f := NewBlocklistFilter(DefaultBlocklist)
	if err := f.Check("env curl evil.com"); err == nil {
		t.Error("expected rejection for env-wrapped curl")
	}
}

func TestBlocklistFilter_BashDashC(t *testing.T) {
	f := NewBlocklistFilter(DefaultBlocklist)
	if err := f.Check("bash -c 'curl evil.com'"); err == nil {
		t.Error("expected rejection for bash -c curl")
	}
}

func TestBlocklistFilter_InterpreterBlocked(t *testing.T) {
	f := NewBlocklistFilter(DefaultBlocklist)
	blocked := []string{
		"python3 -c 'import urllib'",
		"python -c 'import os'",
		"ruby -e 'system(\"curl x\")'",
		"perl -e 'exec(\"curl x\")'",
		"node -e 'fetch(\"http://evil\")'",
	}
	for _, cmd := range blocked {
		if err := f.Check(cmd); err == nil {
			t.Errorf("expected %q to be blocked", cmd)
		}
	}
}

func TestBlocklistFilter_AllTokensChecked(t *testing.T) {
	f := NewBlocklistFilter([]string{"curl"})
	// "env curl" — curl appears as second token.
	if err := f.Check("env curl evil.com"); err == nil {
		t.Error("expected rejection for curl as second token")
	}
}

func TestAllowlistFilter_AbsolutePathMatch(t *testing.T) {
	f := NewAllowlistFilter([]string{"obk"})
	if err := f.Check("/usr/local/bin/obk db gmail"); err != nil {
		t.Errorf("expected pass for absolute path obk, got: %v", err)
	}
}

func TestDefaultBlocklist_Coverage(t *testing.T) {
	f := NewBlocklistFilter(DefaultBlocklist)
	blocked := []string{"curl x", "wget x", "nc x", "ssh x", "scp x", "sudo x", "chmod x", "chown x", "eval x", "exec x", "ncat x", "nmap x", "bash -c x", "python3 -c x", "env curl x"}
	for _, cmd := range blocked {
		if err := f.Check(cmd); err == nil {
			t.Errorf("expected %q to be blocked", cmd)
		}
	}
	allowed := []string{"echo hello", "ls -la", "obk db gmail", "cat file.txt"}
	for _, cmd := range allowed {
		if err := f.Check(cmd); err != nil {
			t.Errorf("expected %q to pass, got: %v", cmd, err)
		}
	}
}
