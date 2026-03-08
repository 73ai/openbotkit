package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/priyanshujain/openbotkit/internal/skills"
)

func TestLoadSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	// Create test skill.
	skillDir := filepath.Join(tmp, "skills", "test-skill")
	if err := os.MkdirAll(skillDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test-skill\n---\nTest content"), 0600); err != nil {
		t.Fatal(err)
	}

	tool := &LoadSkillsTool{}
	input, _ := json.Marshal(map[string]any{"names": []string{"test-skill"}})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "Test content") {
		t.Errorf("result missing skill content: %q", result)
	}
	if !strings.Contains(result, "test-skill") {
		t.Errorf("result missing skill name: %q", result)
	}
}

func TestLoadSkillsMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	if err := os.MkdirAll(filepath.Join(tmp, "skills"), 0700); err != nil {
		t.Fatal(err)
	}

	tool := &LoadSkillsTool{}
	input, _ := json.Marshal(map[string]any{"names": []string{"nonexistent"}})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "not found") {
		t.Errorf("expected 'not found' in result: %q", result)
	}
}

func TestSearchSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	// Create an index.
	idx := &skills.Index{
		Skills: []skills.IndexEntry{
			{Name: "email-read", Description: "Search and read emails from Gmail inbox"},
			{Name: "whatsapp-read", Description: "Search WhatsApp messages"},
			{Name: "memory-read", Description: "Recall past conversations"},
		},
	}
	if err := skills.SaveIndex(idx); err != nil {
		t.Fatal(err)
	}

	tool := &SearchSkillsTool{}
	input, _ := json.Marshal(map[string]string{"query": "email"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "email-read") {
		t.Errorf("expected email-read in results: %q", result)
	}
	if strings.Contains(result, "memory-read") {
		t.Errorf("unexpected memory-read in results: %q", result)
	}
}

func TestSearchSkillsNoMatch(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	idx := &skills.Index{
		Skills: []skills.IndexEntry{
			{Name: "email-read", Description: "Read emails"},
		},
	}
	if err := skills.SaveIndex(idx); err != nil {
		t.Fatal(err)
	}

	tool := &SearchSkillsTool{}
	input, _ := json.Marshal(map[string]string{"query": "calendar"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "No matching skills found." {
		t.Errorf("result = %q", result)
	}
}
