package client

// MockElement represents a minimal DOM element for JS instrumentation execution.
type MockElement struct {
	TagName          string
	ParentNode       *MockElement
	Children         []*MockElement
	LastElementChild *MockElement
	Attributes       map[string]string
}

func NewMockElement(tagName string) *MockElement {
	return &MockElement{
		TagName:    tagName,
		Attributes: make(map[string]string),
	}
}

func (e *MockElement) AppendChild(child *MockElement) *MockElement {
	child.ParentNode = e
	e.Children = append(e.Children, child)
	e.LastElementChild = child
	return child
}

func (e *MockElement) Remove() {
	if e.ParentNode == nil {
		return
	}
	p := e.ParentNode
	for i, c := range p.Children {
		if c == e {
			p.Children = append(p.Children[:i], p.Children[i+1:]...)
			break
		}
	}
	if len(p.Children) > 0 {
		p.LastElementChild = p.Children[len(p.Children)-1]
	} else {
		p.LastElementChild = nil
	}
	e.ParentNode = nil
}

func (e *MockElement) RemoveChild(child *MockElement) *MockElement {
	child.Remove()
	return child
}

func (e *MockElement) SetAttribute(name, value string) {
	e.Attributes[name] = value
}

// MockDocument provides a minimal document object for goja JS execution.
type MockDocument struct {
	elements map[string][]*MockElement
}

func NewMockDocument() *MockDocument {
	head := NewMockElement("head")
	body := NewMockElement("body")
	return &MockDocument{
		elements: map[string][]*MockElement{
			"head": {head},
			"body": {body},
		},
	}
}

func (d *MockDocument) CreateElement(tagName string) *MockElement {
	el := NewMockElement(tagName)
	d.elements[tagName] = append(d.elements[tagName], el)
	return el
}

func (d *MockDocument) GetElementsByTagName(tagName string) []*MockElement {
	return d.elements[tagName]
}
