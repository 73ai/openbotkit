package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/service/learnings"
)

func newTestStore(t *testing.T) *learnings.Store {
	t.Helper()
	return learnings.New(t.TempDir())
}

func TestLearningSaveTool(t *testing.T) {
	st := newTestStore(t)
	tool := NewLearningSaveTool(LearningsDeps{Store: st})

	input, _ := json.Marshal(learningSaveInput{
		Topic:   "Go",
		Bullets: []string{"goroutines are lightweight", "channels for communication"},
	})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "2 bullet") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestLearningSaveToolMissingTopic(t *testing.T) {
	st := newTestStore(t)
	tool := NewLearningSaveTool(LearningsDeps{Store: st})

	input, _ := json.Marshal(learningSaveInput{Bullets: []string{"bullet"}})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for missing topic")
	}
}

func TestLearningSaveToolMissingBullets(t *testing.T) {
	st := newTestStore(t)
	tool := NewLearningSaveTool(LearningsDeps{Store: st})

	input, _ := json.Marshal(learningSaveInput{Topic: "Go"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for missing bullets")
	}
}

func TestLearningReadToolListTopics(t *testing.T) {
	st := newTestStore(t)
	deps := LearningsDeps{Store: st}

	st.Save("Go", []string{"bullet"})
	st.Save("SQL", []string{"bullet"})

	tool := NewLearningReadTool(deps)
	input, _ := json.Marshal(learningReadInput{})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "go") || !strings.Contains(result, "sql") {
		t.Errorf("expected both topics, got: %s", result)
	}
}

func TestLearningReadToolReadTopic(t *testing.T) {
	st := newTestStore(t)
	deps := LearningsDeps{Store: st}

	st.Save("Go", []string{"goroutines are lightweight"})

	tool := NewLearningReadTool(deps)
	input, _ := json.Marshal(learningReadInput{Topic: "Go"})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "goroutines") {
		t.Errorf("expected content, got: %s", result)
	}
}

func TestLearningReadToolEmpty(t *testing.T) {
	st := newTestStore(t)
	tool := NewLearningReadTool(LearningsDeps{Store: st})

	input, _ := json.Marshal(learningReadInput{})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "No learning topics") {
		t.Errorf("expected empty message, got: %s", result)
	}
}

func TestLearningSearchTool(t *testing.T) {
	st := newTestStore(t)
	deps := LearningsDeps{Store: st}

	st.Save("Go", []string{"goroutines are lightweight"})
	st.Save("SQL", []string{"use indexes for speed"})

	tool := NewLearningSearchTool(deps)
	input, _ := json.Marshal(learningSearchInput{Query: "goroutine"})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "go") {
		t.Errorf("expected Go topic in results, got: %s", result)
	}
}

func TestLearningSearchToolNoResults(t *testing.T) {
	st := newTestStore(t)
	deps := LearningsDeps{Store: st}

	st.Save("Go", []string{"goroutines"})

	tool := NewLearningSearchTool(deps)
	input, _ := json.Marshal(learningSearchInput{Query: "kubernetes"})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "No results") {
		t.Errorf("expected no results message, got: %s", result)
	}
}

func TestLearningSaveCreatesFile(t *testing.T) {
	dir := t.TempDir()
	st := learnings.New(dir)
	tool := NewLearningSaveTool(LearningsDeps{Store: st})

	input, _ := json.Marshal(learningSaveInput{
		Topic:   "Docker",
		Bullets: []string{"multi-stage builds reduce image size"},
	})

	tool.Execute(context.Background(), input)

	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	if len(files) == 0 {
		t.Error("expected .md file to be created")
	}

	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "multi-stage") {
		t.Error("expected file to contain bullet content")
	}
}
