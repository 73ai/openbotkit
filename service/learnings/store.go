package learnings

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

type SearchResult struct {
	Topic string
	Line  string
}

type Store struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) Init() error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("create learnings dir: %w", err)
	}
	gitDir := filepath.Join(s.dir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = s.dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %s: %w", out, err)
	}
	s.ensureGitUser()
	return nil
}

func (s *Store) ensureGitUser() {
	for _, key := range []string{"user.name", "user.email"} {
		check := exec.Command("git", "config", key)
		check.Dir = s.dir
		if out, err := check.Output(); err == nil && len(out) > 0 {
			continue
		}
		val := "openbotkit"
		if key == "user.email" {
			val = "bot@local"
		}
		set := exec.Command("git", "config", key, val)
		set.Dir = s.dir
		set.Run()
	}
}

func (s *Store) Save(topic string, bullets []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Init(); err != nil {
		return err
	}

	slug := s.Slug(topic)
	path := filepath.Join(s.dir, slug+".md")

	var content string
	existing, err := os.ReadFile(path)
	if err == nil {
		content = strings.TrimRight(string(existing), "\n") + "\n"
		for _, b := range bullets {
			content += "- " + b + "\n"
		}
	} else {
		content = "# " + topic + "\n\n"
		for _, b := range bullets {
			content += "- " + b + "\n"
		}
	}

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return s.gitCommit(slug+".md", "learn: "+topic)
}

func (s *Store) Read(topic string) (string, error) {
	if strings.Contains(topic, "..") || strings.ContainsAny(topic, "/\\") {
		return "", fmt.Errorf("invalid topic name %q", topic)
	}
	slug := s.Slug(topic)
	path := filepath.Join(s.dir, slug+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("topic %q not found", topic)
	}
	return string(data), nil
}

func (s *Store) List() ([]string, error) {
	entries, err := filepath.Glob(filepath.Join(s.dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("list learnings: %w", err)
	}
	var topics []string
	for _, e := range entries {
		name := s.readTitle(e)
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(e), ".md")
		}
		topics = append(topics, name)
	}
	return topics, nil
}

func (s *Store) readTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	first, _, _ := strings.Cut(string(data), "\n")
	if strings.HasPrefix(first, "# ") {
		return strings.TrimPrefix(first, "# ")
	}
	return ""
}

func (s *Store) Search(query string) ([]SearchResult, error) {
	entries, err := filepath.Glob(filepath.Join(s.dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("search learnings: %w", err)
	}
	lower := strings.ToLower(query)
	var results []SearchResult
	for _, e := range entries {
		data, err := os.ReadFile(e)
		if err != nil {
			continue
		}
		topic := strings.TrimSuffix(filepath.Base(e), ".md")
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(line, "- ") {
				continue
			}
			if strings.Contains(strings.ToLower(line), lower) {
				results = append(results, SearchResult{Topic: topic, Line: line})
			}
		}
	}
	return results, nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Store) Slug(topic string) string {
	lower := strings.ToLower(strings.TrimFunc(topic, unicode.IsSpace))
	slug := slugRe.ReplaceAllString(lower, "-")
	return strings.Trim(slug, "-")
}

func (s *Store) gitCommit(filename, message string) error {
	add := exec.Command("git", "add", filename)
	add.Dir = s.dir
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s: %w", out, err)
	}

	commit := exec.Command("git", "commit", "-m", message)
	commit.Dir = s.dir
	if out, err := commit.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %s: %w", out, err)
	}
	return nil
}
