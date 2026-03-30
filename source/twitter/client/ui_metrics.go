package client

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

var (
	jsonStringifyRe = regexp.MustCompile(`JSON\.stringify\((\w+)\(\)\)`)
	eqFixRe         = regexp.MustCompile(`\(!\([a-zA-Z]{5}\)\|\|[a-zA-Z]{5}\)==[a-zA-Z]{5}`)
)

// SolveUIMetrics executes X's JS instrumentation challenge and returns the response string.
func SolveUIMetrics(jsCode string) (string, error) {
	funcBody := extractComputationFunc(jsCode)
	if funcBody == "" {
		return "", fmt.Errorf("could not extract computation function from JS instrumentation")
	}

	funcBody = fixEqualityOperators(funcBody)

	vm := goja.New()
	cache := &jsCache{vm: vm, objects: make(map[*MockElement]*goja.Object)}
	if err := injectMockDOM(vm, cache); err != nil {
		return "", fmt.Errorf("inject mock DOM: %w", err)
	}

	wrapped := "(function() " + funcBody + ")()"
	val, err := vm.RunString(wrapped)
	if err != nil {
		return "", fmt.Errorf("execute JS instrumentation: %w", err)
	}

	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return "", fmt.Errorf("JS instrumentation returned nil")
	}

	result := val.Export()
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal instrumentation result: %w", err)
	}

	return string(jsonBytes), nil
}

// extractComputationFunc finds the inner computation function that gets
// JSON.stringified and returns its body (including braces).
func extractComputationFunc(js string) string {
	m := jsonStringifyRe.FindStringSubmatch(js)
	if len(m) < 2 {
		return extractFunctionBody(js)
	}

	funcName := m[1]
	prefix := "function " + funcName + "()"
	idx := strings.Index(js, prefix)
	if idx < 0 {
		return extractFunctionBody(js)
	}

	braceStart := strings.Index(js[idx:], "{")
	if braceStart < 0 {
		return extractFunctionBody(js)
	}
	braceStart += idx

	depth := 0
	for i := braceStart; i < len(js); i++ {
		switch js[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return js[braceStart : i+1]
			}
		}
	}

	return extractFunctionBody(js)
}

var simpleFuncBodyRe = regexp.MustCompile(`(?s)function\s+[a-zA-Z]+\(\)\s*(\{.+\})`)

func extractFunctionBody(js string) string {
	m := simpleFuncBodyRe.FindStringSubmatch(js)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

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

// jsCache maintains a mapping from MockElement to goja Object so that
// JS identity comparisons (el1 == el2) work correctly.
type jsCache struct {
	vm      *goja.Runtime
	objects map[*MockElement]*goja.Object
}

func (c *jsCache) getOrCreate(el *MockElement) *goja.Object {
	if obj, ok := c.objects[el]; ok {
		return obj
	}
	obj := c.vm.NewObject()
	c.objects[el] = obj
	bindElement(c, obj, el)
	return obj
}

func injectMockDOM(vm *goja.Runtime, cache *jsCache) error {
	doc := NewMockDocument()

	docObj := vm.NewObject()

	docObj.Set("createElement", func(call goja.FunctionCall) goja.Value {
		tagName := call.Argument(0).String()
		el := doc.CreateElement(tagName)
		return cache.getOrCreate(el)
	})

	docObj.Set("getElementsByTagName", func(call goja.FunctionCall) goja.Value {
		tagName := call.Argument(0).String()
		elements := doc.GetElementsByTagName(tagName)
		arr := vm.NewArray()
		for i, el := range elements {
			arr.Set(fmt.Sprintf("%d", i), cache.getOrCreate(el))
		}
		arr.Set("length", len(elements))
		return arr
	})

	vm.Set("document", docObj)
	return nil
}

func bindElement(cache *jsCache, obj *goja.Object, el *MockElement) {
	vm := cache.vm

	obj.Set("tagName", el.TagName)

	obj.Set("appendChild", func(call goja.FunctionCall) goja.Value {
		childObj := call.Argument(0).ToObject(vm)
		childEl := nativeElement(childObj)
		if childEl != nil {
			el.AppendChild(childEl)
		}
		return call.Argument(0)
	})

	obj.Set("removeChild", func(call goja.FunctionCall) goja.Value {
		childObj := call.Argument(0).ToObject(vm)
		childEl := nativeElement(childObj)
		if childEl != nil {
			el.RemoveChild(childEl)
		}
		return call.Argument(0)
	})

	obj.Set("remove", func(call goja.FunctionCall) goja.Value {
		el.Remove()
		return goja.Undefined()
	})

	obj.Set("setAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		value := call.Argument(1).String()
		el.SetAttribute(name, value)
		return goja.Undefined()
	})

	obj.Set("_native", el)

	defineProperties(cache, obj, el)
}

func defineProperties(cache *jsCache, obj *goja.Object, el *MockElement) {
	vm := cache.vm

	obj.Set("_getChildren", func(call goja.FunctionCall) goja.Value {
		arr := vm.NewArray()
		for i, child := range el.Children {
			arr.Set(fmt.Sprintf("%d", i), cache.getOrCreate(child))
		}
		arr.Set("length", len(el.Children))
		return arr
	})

	obj.Set("_getLastElementChild", func(call goja.FunctionCall) goja.Value {
		if el.LastElementChild == nil {
			return goja.Null()
		}
		return cache.getOrCreate(el.LastElementChild)
	})

	obj.Set("_getParentNode", func(call goja.FunctionCall) goja.Value {
		if el.ParentNode == nil {
			return goja.Null()
		}
		return cache.getOrCreate(el.ParentNode)
	})

	obj.Set("_getInnerText", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(el.InnerText)
	})

	obj.Set("_setInnerText", func(call goja.FunctionCall) goja.Value {
		el.InnerText = call.Argument(0).String()
		return goja.Undefined()
	})

	vm.Set("__tempObj", obj)
	vm.RunString(`
		(function(o) {
			Object.defineProperty(o, 'children', {
				get: function() { return o._getChildren(); },
				configurable: true
			});
			Object.defineProperty(o, 'lastElementChild', {
				get: function() { return o._getLastElementChild(); },
				configurable: true
			});
			Object.defineProperty(o, 'parentNode', {
				get: function() { return o._getParentNode(); },
				configurable: true
			});
			Object.defineProperty(o, 'innerText', {
				get: function() { return o._getInnerText(); },
				set: function(v) { o._setInnerText(v); },
				configurable: true
			});
		})(__tempObj);
	`)
	vm.Set("__tempObj", goja.Undefined())
}

func nativeElement(obj *goja.Object) *MockElement {
	v := obj.Get("_native")
	if v == nil {
		return nil
	}
	el, _ := v.Export().(*MockElement)
	return el
}
