package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DirExploreTool lists or searches files and directories. Pure Go, no shell.
type DirExploreTool struct{}

func (d *DirExploreTool) Name() string        { return "dir_explore" }
func (d *DirExploreTool) Description() string { return "List or search files and directories" }
func (d *DirExploreTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["ls", "tree", "find", "glob"], "description": "Action to perform"},
			"path": {"type": "string", "description": "Directory path to explore"},
			"pattern": {"type": "string", "description": "Pattern for find/glob actions"},
			"max_depth": {"type": "integer", "description": "Max depth (default 3)"}
		},
		"required": ["action", "path"]
	}`)
}

type dirExploreInput struct {
	Action   string `json:"action"`
	Path     string `json:"path"`
	Pattern  string `json:"pattern"`
	MaxDepth int    `json:"max_depth"`
}

func (d *DirExploreTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var in dirExploreInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.MaxDepth <= 0 {
		in.MaxDepth = 3
	}

	switch in.Action {
	case "ls":
		return dirExploreLs(in.Path)
	case "tree":
		return dirExploreTree(in.Path, in.MaxDepth)
	case "find":
		return dirExploreFind(in.Path, in.Pattern, in.MaxDepth)
	case "glob":
		return dirExploreGlob(in.Path, in.Pattern)
	default:
		return "", fmt.Errorf("unknown action %q (use ls, tree, find, or glob)", in.Action)
	}
}

func dirExploreLs(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("read dir: %w", err)
	}
	var b strings.Builder
	for _, e := range entries {
		suffix := ""
		if e.IsDir() {
			suffix = "/"
		}
		fmt.Fprintf(&b, "%s%s\n", e.Name(), suffix)
	}
	result := b.String()
	return TruncateHead(result, MaxLinesFileRead), nil
}

func dirExploreTree(root string, maxDepth int) (string, error) {
	var b strings.Builder
	baseDepth := strings.Count(filepath.Clean(root), string(os.PathSeparator))
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		depth := strings.Count(filepath.Clean(path), string(os.PathSeparator)) - baseDepth
		if depth > maxDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		indent := strings.Repeat("  ", depth)
		suffix := ""
		if d.IsDir() && path != root {
			suffix = "/"
		}
		fmt.Fprintf(&b, "%s%s%s\n", indent, d.Name(), suffix)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk: %w", err)
	}
	result := b.String()
	return TruncateHead(result, MaxLinesFileRead), nil
}

func dirExploreFind(root, pattern string, maxDepth int) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("pattern is required for find action")
	}
	var b strings.Builder
	baseDepth := strings.Count(filepath.Clean(root), string(os.PathSeparator))
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		depth := strings.Count(filepath.Clean(path), string(os.PathSeparator)) - baseDepth
		if depth > maxDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		matched, _ := filepath.Match(pattern, d.Name())
		if matched {
			fmt.Fprintln(&b, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk: %w", err)
	}
	result := b.String()
	return TruncateHead(result, MaxLinesFileRead), nil
}

func dirExploreGlob(root, pattern string) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("pattern is required for glob action")
	}
	full := filepath.Join(root, pattern)
	matches, err := filepath.Glob(full)
	if err != nil {
		return "", fmt.Errorf("glob: %w", err)
	}
	var b strings.Builder
	for _, m := range matches {
		fmt.Fprintln(&b, m)
	}
	result := b.String()
	return TruncateHead(result, MaxLinesFileRead), nil
}
