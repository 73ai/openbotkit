package client

import "testing"

func TestExtractFunctionBody(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple function",
			input: `function abc() {return "hello"}`,
			want:  `{return "hello"}`,
		},
		{
			name:  "function with complex body",
			input: `var x=1;function myFunc() {var a=1;return a+2}; other stuff`,
			want:  `{var a=1;return a+2}`,
		},
		{
			name:  "no function",
			input: `var x = 1;`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFunctionBody(tt.input)
			if got != tt.want {
				t.Errorf("extractFunctionBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFixEqualityOperators(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "matching pattern gets fixed",
			input: `(!(abcde)||fghij)==klmno`,
			want:  `(!(abcde)||fghij)===klmno`,
		},
		{
			name:  "already triple equals unchanged",
			input: `(!(abcde)||fghij)===klmno`,
			want:  `(!(abcde)||fghij)===klmno`,
		},
		{
			name:  "non-matching pattern unchanged",
			input: `x==y`,
			want:  `x==y`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fixEqualityOperators(tt.input)
			if got != tt.want {
				t.Errorf("fixEqualityOperators() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSolveUIMetrics_SimpleJS(t *testing.T) {
	js := `function solve() {return "test_result"}`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}
	if result != "test_result" {
		t.Errorf("SolveUIMetrics() = %q, want %q", result, "test_result")
	}
}

func TestSolveUIMetrics_WithDocumentAccess(t *testing.T) {
	js := `function solve() {
		var el = document.createElement("div");
		el.setAttribute("id", "test");
		var heads = document.getElementsByTagName("head");
		if (heads.length > 0) {
			heads[0].appendChild(el);
		}
		return "instrumented";
	}`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}
	if result != "instrumented" {
		t.Errorf("SolveUIMetrics() = %q, want %q", result, "instrumented")
	}
}

func TestSolveUIMetrics_NoFunction(t *testing.T) {
	_, err := SolveUIMetrics("var x = 1;")
	if err == nil {
		t.Error("expected error for input without function")
	}
}

func TestSolveUIMetrics_StringConcat(t *testing.T) {
	js := `function metrics() {var a = "rf"; var b = "123"; return a + b;}`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}
	if result != "rf123" {
		t.Errorf("SolveUIMetrics() = %q, want %q", result, "rf123")
	}
}

func TestSolveUIMetrics_ElementRemove(t *testing.T) {
	js := `function metrics() {
		var el = document.createElement("script");
		var head = document.getElementsByTagName("head")[0];
		head.appendChild(el);
		el.remove();
		return "cleaned";
	}`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}
	if result != "cleaned" {
		t.Errorf("SolveUIMetrics() = %q, want %q", result, "cleaned")
	}
}
