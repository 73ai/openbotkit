package slack

import (
	"encoding/json"
	"testing"
)

func TestCompactJSON_StripNulls(t *testing.T) {
	input := `{"name":"alice","age":null,"email":""}`
	got, err := CompactJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(got, &m)
	if _, ok := m["age"]; ok {
		t.Error("null field should be stripped")
	}
	if _, ok := m["email"]; ok {
		t.Error("empty string should be stripped")
	}
	if m["name"] != "alice" {
		t.Errorf("name = %v", m["name"])
	}
}

func TestCompactJSON_StripEmptyArraysAndObjects(t *testing.T) {
	input := `{"items":[],"meta":{},"name":"test"}`
	got, err := CompactJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(got, &m)
	if _, ok := m["items"]; ok {
		t.Error("empty array should be stripped")
	}
	if _, ok := m["meta"]; ok {
		t.Error("empty object should be stripped")
	}
}

func TestCompactJSON_NestedCompaction(t *testing.T) {
	input := `{"outer":{"inner":null,"val":"keep"}}`
	got, err := CompactJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(got, &m)
	outer := m["outer"].(map[string]any)
	if _, ok := outer["inner"]; ok {
		t.Error("nested null should be stripped")
	}
	if outer["val"] != "keep" {
		t.Errorf("nested val = %v", outer["val"])
	}
}

func TestCompactJSON_PreservesNonEmpty(t *testing.T) {
	input := `{"a":1,"b":"hello","c":[1,2],"d":true}`
	got, err := CompactJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(got, &m)
	if len(m) != 4 {
		t.Errorf("expected 4 keys, got %d: %v", len(m), m)
	}
}

func TestCompactJSON_StripZero(t *testing.T) {
	input := `{"count":0,"active":false,"name":"test"}`
	got, err := CompactJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(got, &m)
	if _, ok := m["count"]; ok {
		t.Error("zero number should be stripped")
	}
	if _, ok := m["active"]; ok {
		t.Error("false bool should be stripped")
	}
}
