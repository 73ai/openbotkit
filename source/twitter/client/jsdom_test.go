package client

import "testing"

func TestMockElement_AppendChild(t *testing.T) {
	parent := NewMockElement("div")
	child1 := NewMockElement("span")
	child2 := NewMockElement("p")

	parent.AppendChild(child1)
	parent.AppendChild(child2)

	if len(parent.Children) != 2 {
		t.Fatalf("Children count = %d, want 2", len(parent.Children))
	}
	if parent.LastElementChild != child2 {
		t.Error("LastElementChild should be child2")
	}
	if child1.ParentNode != parent {
		t.Error("child1.ParentNode should be parent")
	}
}

func TestMockElement_Remove(t *testing.T) {
	parent := NewMockElement("div")
	child1 := NewMockElement("span")
	child2 := NewMockElement("p")
	parent.AppendChild(child1)
	parent.AppendChild(child2)

	child2.Remove()

	if len(parent.Children) != 1 {
		t.Fatalf("Children count = %d, want 1", len(parent.Children))
	}
	if parent.LastElementChild != child1 {
		t.Error("LastElementChild should be child1 after removing child2")
	}
	if child2.ParentNode != nil {
		t.Error("removed child ParentNode should be nil")
	}
}

func TestMockElement_RemoveLastChild(t *testing.T) {
	parent := NewMockElement("div")
	child := NewMockElement("span")
	parent.AppendChild(child)

	child.Remove()

	if len(parent.Children) != 0 {
		t.Fatalf("Children count = %d, want 0", len(parent.Children))
	}
	if parent.LastElementChild != nil {
		t.Error("LastElementChild should be nil")
	}
}

func TestMockElement_RemoveChild(t *testing.T) {
	parent := NewMockElement("div")
	child := NewMockElement("span")
	parent.AppendChild(child)

	result := parent.RemoveChild(child)

	if result != child {
		t.Error("RemoveChild should return the removed child")
	}
	if len(parent.Children) != 0 {
		t.Fatalf("Children count = %d, want 0", len(parent.Children))
	}
}

func TestMockElement_SetAttribute(t *testing.T) {
	el := NewMockElement("div")
	el.SetAttribute("id", "test")
	el.SetAttribute("class", "foo")

	if el.Attributes["id"] != "test" {
		t.Errorf("Attribute id = %q, want test", el.Attributes["id"])
	}
	if el.Attributes["class"] != "foo" {
		t.Errorf("Attribute class = %q, want foo", el.Attributes["class"])
	}
}

func TestMockElement_RemoveOrphan(t *testing.T) {
	el := NewMockElement("div")
	el.Remove() // should not panic
}

func TestMockDocument_CreateElement(t *testing.T) {
	doc := NewMockDocument()
	el := doc.CreateElement("script")

	if el.TagName != "script" {
		t.Errorf("TagName = %q, want script", el.TagName)
	}
	scripts := doc.GetElementsByTagName("script")
	if len(scripts) != 1 || scripts[0] != el {
		t.Error("GetElementsByTagName should return the created element")
	}
}

func TestMockDocument_GetElementsByTagName(t *testing.T) {
	doc := NewMockDocument()

	heads := doc.GetElementsByTagName("head")
	if len(heads) != 1 {
		t.Fatalf("head elements = %d, want 1", len(heads))
	}

	bodies := doc.GetElementsByTagName("body")
	if len(bodies) != 1 {
		t.Fatalf("body elements = %d, want 1", len(bodies))
	}

	divs := doc.GetElementsByTagName("div")
	if len(divs) != 0 {
		t.Fatalf("div elements = %d, want 0", len(divs))
	}
}
