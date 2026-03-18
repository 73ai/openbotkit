package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupDirExploreTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0755)
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "nested.go"), []byte("package sub"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "deep", "deep.txt"), []byte("deep"), 0644)
	return dir
}

func TestDirExplore_Ls(t *testing.T) {
	dir := setupDirExploreTestDir(t)
	tool := &DirExploreTool{}
	input, _ := json.Marshal(dirExploreInput{Action: "ls", Path: dir})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "hello.txt") {
		t.Error("expected hello.txt in ls output")
	}
	if !strings.Contains(out, "sub/") {
		t.Error("expected sub/ in ls output")
	}
}

func TestDirExplore_Tree(t *testing.T) {
	dir := setupDirExploreTestDir(t)
	tool := &DirExploreTool{}
	input, _ := json.Marshal(dirExploreInput{Action: "tree", Path: dir, MaxDepth: 2})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "nested.go") {
		t.Error("expected nested.go in tree output")
	}
}

func TestDirExplore_Tree_DepthLimit(t *testing.T) {
	dir := setupDirExploreTestDir(t)
	tool := &DirExploreTool{}
	input, _ := json.Marshal(dirExploreInput{Action: "tree", Path: dir, MaxDepth: 1})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out, "deep.txt") {
		t.Error("deep.txt should not appear at depth 1")
	}
}

func TestDirExplore_Find(t *testing.T) {
	dir := setupDirExploreTestDir(t)
	tool := &DirExploreTool{}
	input, _ := json.Marshal(dirExploreInput{Action: "find", Path: dir, Pattern: "*.go"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "main.go") {
		t.Error("expected main.go in find output")
	}
	if !strings.Contains(out, "nested.go") {
		t.Error("expected nested.go in find output")
	}
	if strings.Contains(out, "hello.txt") {
		t.Error("hello.txt should not match *.go")
	}
}

func TestDirExplore_Glob(t *testing.T) {
	dir := setupDirExploreTestDir(t)
	tool := &DirExploreTool{}
	input, _ := json.Marshal(dirExploreInput{Action: "glob", Path: dir, Pattern: "*.txt"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "hello.txt") {
		t.Error("expected hello.txt in glob output")
	}
}

func TestDirExplore_FindNoPattern(t *testing.T) {
	dir := setupDirExploreTestDir(t)
	tool := &DirExploreTool{}
	input, _ := json.Marshal(dirExploreInput{Action: "find", Path: dir})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for find without pattern")
	}
}

func TestDirExplore_UnknownAction(t *testing.T) {
	tool := &DirExploreTool{}
	input, _ := json.Marshal(dirExploreInput{Action: "bad", Path: "."})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for unknown action")
	}
}
