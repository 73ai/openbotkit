package client

import (
	"fmt"
	"regexp"

	"github.com/dop251/goja"
)

var (
	funcBodyRe = regexp.MustCompile(`(?s)function\s+[a-zA-Z]+\(\)\s*(\{.+\})`)
	eqFixRe    = regexp.MustCompile(`\(!\([a-zA-Z]{5}\)\|\|[a-zA-Z]{5}\)==[a-zA-Z]{5}`)
)

// SolveUIMetrics executes X's JS instrumentation challenge and returns the response string.
func SolveUIMetrics(jsCode string) (string, error) {
	body := extractFunctionBody(jsCode)
	if body == "" {
		return "", fmt.Errorf("could not extract function body from JS instrumentation")
	}

	body = fixEqualityOperators(body)

	vm := goja.New()
	if err := injectMockDOM(vm); err != nil {
		return "", fmt.Errorf("inject mock DOM: %w", err)
	}

	wrapped := "(function() " + body + ")()"
	val, err := vm.RunString(wrapped)
	if err != nil {
		return "", fmt.Errorf("execute JS instrumentation: %w", err)
	}

	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return "", fmt.Errorf("JS instrumentation returned nil")
	}

	return val.String(), nil
}

func extractFunctionBody(js string) string {
	m := funcBodyRe.FindStringSubmatch(js)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// fixEqualityOperators replaces == with === for the specific pattern X uses,
// matching twikit's approach to fix JS strict equality checks.
func fixEqualityOperators(js string) string {
	return eqFixRe.ReplaceAllStringFunc(js, func(match string) string {
		return replaceNonTripleEquals(match)
	})
}

func replaceNonTripleEquals(s string) string {
	result := make([]byte, 0, len(s)+1)
	for i := 0; i < len(s); i++ {
		if s[i] == '=' && i+1 < len(s) && s[i+1] == '=' {
			if i+2 < len(s) && s[i+2] == '=' {
				result = append(result, '=', '=', '=')
				i += 2
			} else {
				result = append(result, '=', '=', '=')
				i += 1
			}
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}

func injectMockDOM(vm *goja.Runtime) error {
	doc := NewMockDocument()

	docObj := vm.NewObject()

	docObj.Set("createElement", func(call goja.FunctionCall) goja.Value {
		tagName := call.Argument(0).String()
		el := doc.CreateElement(tagName)
		return elementToJS(vm, el)
	})

	docObj.Set("getElementsByTagName", func(call goja.FunctionCall) goja.Value {
		tagName := call.Argument(0).String()
		elements := doc.GetElementsByTagName(tagName)
		arr := vm.NewArray()
		for i, el := range elements {
			arr.Set(fmt.Sprintf("%d", i), elementToJS(vm, el))
		}
		arr.Set("length", len(elements))
		return arr
	})

	vm.Set("document", docObj)
	return nil
}

func elementToJS(vm *goja.Runtime, el *MockElement) *goja.Object {
	obj := vm.NewObject()
	obj.Set("tagName", el.TagName)

	obj.Set("appendChild", func(call goja.FunctionCall) goja.Value {
		childObj := call.Argument(0).ToObject(vm)
		childEl, ok := childObj.Get("_native").Export().(*MockElement)
		if ok {
			el.AppendChild(childEl)
		}
		return call.Argument(0)
	})

	obj.Set("remove", func(call goja.FunctionCall) goja.Value {
		el.Remove()
		return goja.Undefined()
	})

	obj.Set("removeChild", func(call goja.FunctionCall) goja.Value {
		childObj := call.Argument(0).ToObject(vm)
		childEl, ok := childObj.Get("_native").Export().(*MockElement)
		if ok {
			el.RemoveChild(childEl)
		}
		return call.Argument(0)
	})

	obj.Set("setAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		value := call.Argument(1).String()
		el.SetAttribute(name, value)
		return goja.Undefined()
	})

	// Dynamic property for children/lastElementChild since they change
	obj.Set("_native", el)

	defineChildrenProperties(vm, obj, el)

	return obj
}

func defineChildrenProperties(vm *goja.Runtime, obj *goja.Object, el *MockElement) {
	// Use defineProperty for dynamic getters
	vm.Set("__tempObj", obj)
	vm.Set("__tempEl", el)

	vm.RunString(`
		Object.defineProperty(__tempObj, 'children', {
			get: function() { return __tempObj._getChildren(); },
			configurable: true
		});
		Object.defineProperty(__tempObj, 'lastElementChild', {
			get: function() { return __tempObj._getLastElementChild(); },
			configurable: true
		});
		Object.defineProperty(__tempObj, 'parentNode', {
			get: function() { return __tempObj._getParentNode(); },
			configurable: true
		});
	`)

	obj.Set("_getChildren", func(call goja.FunctionCall) goja.Value {
		arr := vm.NewArray()
		for i, child := range el.Children {
			arr.Set(fmt.Sprintf("%d", i), elementToJS(vm, child))
		}
		arr.Set("length", len(el.Children))
		return arr
	})

	obj.Set("_getLastElementChild", func(call goja.FunctionCall) goja.Value {
		if el.LastElementChild == nil {
			return goja.Null()
		}
		return elementToJS(vm, el.LastElementChild)
	})

	obj.Set("_getParentNode", func(call goja.FunctionCall) goja.Value {
		if el.ParentNode == nil {
			return goja.Null()
		}
		return elementToJS(vm, el.ParentNode)
	})

	vm.Set("__tempObj", goja.Undefined())
	vm.Set("__tempEl", goja.Undefined())
}
