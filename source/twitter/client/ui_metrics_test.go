package client

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestExtractComputationFunc_WithJSONStringify(t *testing.T) {
	js := `
	function outer() {
		function compute() {var x = 1; return {"a": x};}
		var r = JSON.stringify(compute());
	}
	outer();`

	body := extractComputationFunc(js)
	if !strings.Contains(body, `return {"a": x}`) {
		t.Errorf("expected computation body, got %q", body)
	}
}

func TestExtractComputationFunc_Fallback(t *testing.T) {
	js := `function simple() {return "hello"}`
	body := extractComputationFunc(js)
	if body != `{return "hello"}` {
		t.Errorf("extractComputationFunc() = %q, want {return \"hello\"}", body)
	}
}

func TestExtractComputationFunc_NoFunction(t *testing.T) {
	body := extractComputationFunc("var x = 1;")
	if body != "" {
		t.Errorf("expected empty, got %q", body)
	}
}

func TestReplaceNonTripleEquals_AlreadyTriple(t *testing.T) {
	input := `(!(abcde)||fghij)===klmno`
	got := replaceNonTripleEquals(input)
	if got != `(!(abcde)||fghij)===klmno` {
		t.Errorf("replaceNonTripleEquals() = %q, want unchanged", got)
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
	if result != `"test_result"` {
		t.Errorf("SolveUIMetrics() = %q, want %q", result, `"test_result"`)
	}
}

func TestSolveUIMetrics_ReturnsObject(t *testing.T) {
	js := `function solve() {return {"key": "value"}}`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("parsed[key] = %v, want value", parsed["key"])
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
		return {"status": "ok"};
	}`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}
	if !strings.Contains(result, `"ok"`) {
		t.Errorf("result = %q, expected to contain ok", result)
	}
}

func TestSolveUIMetrics_NoFunction(t *testing.T) {
	_, err := SolveUIMetrics("var x = 1;")
	if err == nil {
		t.Error("expected error for input without function")
	}
}

func TestSolveUIMetrics_InnerText(t *testing.T) {
	js := `function solve() {
		var el = document.createElement("div");
		el.innerText = 42;
		return {"value": parseInt(el.innerText)};
	}`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed["value"] != float64(42) {
		t.Errorf("value = %v, want 42", parsed["value"])
	}
}

func TestSolveUIMetrics_DOMTreeManipulation(t *testing.T) {
	js := `function solve() {
		var root = document.createElement("div");
		document.getElementsByTagName("body")[0].appendChild(root);
		var child1 = document.createElement("div");
		root.appendChild(child1);
		child1.innerText = 10;
		var child2 = document.createElement("div");
		root.appendChild(child2);
		child2.innerText = 20;
		while (root.children.length > 0) {
			root.removeChild(root.lastElementChild);
		}
		root.parentNode.removeChild(root);
		return {"result": 30};
	}`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}
	if !strings.Contains(result, "30") {
		t.Errorf("result = %q, expected 30", result)
	}
}

func TestSolveUIMetrics_ElementIdentity(t *testing.T) {
	js := `function solve() {
		var root = document.createElement("div");
		document.getElementsByTagName("body")[0].appendChild(root);
		var child = document.createElement("div");
		root.appendChild(child);
		var same = (child.parentNode == root) ? "yes" : "no";
		return {"same": same};
	}`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}
	if !strings.Contains(result, `"yes"`) {
		t.Errorf("identity check failed, result = %q", result)
	}
}

func TestSolveUIMetrics_RealXJS(t *testing.T) {
	data, err := os.ReadFile("/tmp/x_js_inst.js")
	if err != nil {
		t.Skip("no cached X JS instrumentation at /tmp/x_js_inst.js")
	}

	result, err := SolveUIMetrics(string(data))
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v\nraw: %s", err, result)
	}

	if _, ok := parsed["rf"]; !ok {
		t.Error("result missing 'rf' key")
	}
	if _, ok := parsed["s"]; !ok {
		t.Error("result missing 's' key")
	}
}

func TestSolveUIMetrics_JSRuntimeError(t *testing.T) {
	js := `function solve() {return nonExistentVar.property;}`
	_, err := SolveUIMetrics(js)
	if err == nil {
		t.Error("expected error for JS runtime error")
	}
	if !strings.Contains(err.Error(), "execute JS instrumentation") {
		t.Errorf("error = %q, want to contain 'execute JS instrumentation'", err.Error())
	}
}

func TestSolveUIMetrics_ReturnsUndefined(t *testing.T) {
	js := `function solve() {var x = 1; return undefined;}`
	_, err := SolveUIMetrics(js)
	if err == nil {
		t.Error("expected error for undefined return")
	}
	if !strings.Contains(err.Error(), "returned nil") {
		t.Errorf("error = %q, want to contain 'returned nil'", err.Error())
	}
}

func TestExtractComputationFunc_FuncNameNotFound(t *testing.T) {
	js := `var r = JSON.stringify(missingFunc()); function other() {return 1}`
	body := extractComputationFunc(js)
	if body != "{return 1}" {
		t.Errorf("expected fallback extraction, got %q", body)
	}
}

func TestExtractComputationFunc_UnbalancedBraces(t *testing.T) {
	js := `var r = JSON.stringify(broken()); function broken() {unclosed`
	body := extractComputationFunc(js)
	if body != "" {
		t.Errorf("expected empty for unbalanced braces, got %q", body)
	}
}

func TestSolveUIMetrics_ExtractFromWrappedJS(t *testing.T) {
	js := `function outer() {
		function compute() {return {"rf": {"a": 1}, "s": "sig"}}
		var r;
		try { r = JSON.stringify(compute()); } catch(e) {}
		return r;
	};
	outer();`

	result, err := SolveUIMetrics(js)
	if err != nil {
		t.Fatalf("SolveUIMetrics() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed["s"] != "sig" {
		t.Errorf("s = %v, want sig", parsed["s"])
	}
}
