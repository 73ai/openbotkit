package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Store manages user memories as Markdown files on disk.
type Store struct {
	dir string
	mu  sync.RWMutex
}

// NewStore creates a Store backed by the given directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

type memoryLine struct {
	ID      int64
	Source  string
	Content string
}

// Matches: - [id|source] content  OR  - [id] content (legacy, source="")
var bulletRe = regexp.MustCompile(`^- \[(\d+)(?:\|([^]]*))?\] (.+)$`)

type counterFile struct {
	NextID int64 `json:"next_id"`
}

func (s *Store) Add(content string, category Category, source, sourceRef string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := s.nextID()
	if err != nil {
		return 0, err
	}

	lines, _ := s.readCategory(category)
	lines = append(lines, memoryLine{ID: id, Source: source, Content: content})
	if err := s.writeCategory(category, lines); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) Get(id int64) (*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, cat := range categoryOrder {
		lines, _ := s.readCategory(cat)
		for _, line := range lines {
			if line.ID == id {
				return &Memory{
					ID:       line.ID,
					Content:  line.Content,
					Category: cat,
					Source:   line.Source,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("get memory: not found (id=%d)", id)
}

func (s *Store) Update(id int64, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, cat := range categoryOrder {
		lines, _ := s.readCategory(cat)
		for i, line := range lines {
			if line.ID == id {
				lines[i].Content = content
				return s.writeCategory(cat, lines)
			}
		}
	}
	return fmt.Errorf("update memory: not found (id=%d)", id)
}

func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, cat := range categoryOrder {
		lines, _ := s.readCategory(cat)
		for i, line := range lines {
			if line.ID == id {
				lines = append(lines[:i], lines[i+1:]...)
				return s.writeCategory(cat, lines)
			}
		}
	}
	return fmt.Errorf("delete memory: not found (id=%d)", id)
}

func (s *Store) List() ([]Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []Memory
	for _, cat := range categoryOrder {
		lines, _ := s.readCategory(cat)
		for _, line := range lines {
			all = append(all, Memory{
				ID:       line.ID,
				Content:  line.Content,
				Category: cat,
				Source:   line.Source,
			})
		}
	}
	return all, nil
}

func (s *Store) ListByCategory(category Category) ([]Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lines, _ := s.readCategory(category)
	var result []Memory
	for _, line := range lines {
		result = append(result, Memory{
			ID:       line.ID,
			Content:  line.Content,
			Category: category,
			Source:   line.Source,
		})
	}
	return result, nil
}

func (s *Store) Search(query string) ([]Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lower := strings.ToLower(query)
	var result []Memory
	for _, cat := range categoryOrder {
		lines, _ := s.readCategory(cat)
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line.Content), lower) {
				result = append(result, Memory{
					ID:       line.ID,
					Content:  line.Content,
					Category: cat,
					Source:   line.Source,
				})
			}
		}
	}
	return result, nil
}

func (s *Store) Count() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	for _, cat := range categoryOrder {
		lines, _ := s.readCategory(cat)
		count += int64(len(lines))
	}
	return count, nil
}

func (s *Store) FormatForPrompt() string {
	memories, _ := s.List()
	return FormatForPrompt(memories)
}

// Internal helpers

func (s *Store) categoryFile(cat Category) string {
	return filepath.Join(s.dir, string(cat)+".md")
}

func (s *Store) readCategory(cat Category) ([]memoryLine, error) {
	data, err := os.ReadFile(s.categoryFile(cat))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var lines []memoryLine
	for _, raw := range strings.Split(string(data), "\n") {
		matches := bulletRe.FindStringSubmatch(raw)
		if matches == nil {
			continue
		}
		id, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			continue
		}
		lines = append(lines, memoryLine{ID: id, Source: matches[2], Content: matches[3]})
	}
	return lines, nil
}

func (s *Store) writeCategory(cat Category, lines []memoryLine) error {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("category: " + string(cat) + "\n")
	sb.WriteString("updated_at: \"" + time.Now().UTC().Format(time.RFC3339) + "\"\n")
	sb.WriteString("---\n\n")
	for _, line := range lines {
		sb.WriteString(fmt.Sprintf("- [%d|%s] %s\n", line.ID, line.Source, line.Content))
	}

	tmpPath := s.categoryFile(cat) + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(sb.String()), 0600); err != nil {
		return fmt.Errorf("write category %s: %w", cat, err)
	}
	if err := os.Rename(tmpPath, s.categoryFile(cat)); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename category %s: %w", cat, err)
	}
	return nil
}

func (s *Store) nextID() (int64, error) {
	counterPath := filepath.Join(s.dir, ".counter")
	var c counterFile

	data, err := os.ReadFile(counterPath)
	if err == nil {
		json.Unmarshal(data, &c)
	}
	if c.NextID == 0 {
		c.NextID = 1
	}

	id := c.NextID
	c.NextID++

	out, _ := json.Marshal(c)
	if err := os.WriteFile(counterPath, out, 0600); err != nil {
		return 0, fmt.Errorf("write counter: %w", err)
	}
	return id, nil
}

// AllCategories returns the list of known categories in display order.
func AllCategories() []Category {
	return append([]Category(nil), categoryOrder...)
}

// SortMemories sorts memories by category order, then by ID.
func SortMemories(memories []Memory) {
	catIdx := make(map[Category]int)
	for i, c := range categoryOrder {
		catIdx[c] = i
	}
	sort.Slice(memories, func(i, j int) bool {
		ci, cj := catIdx[memories[i].Category], catIdx[memories[j].Category]
		if ci != cj {
			return ci < cj
		}
		return memories[i].ID < memories[j].ID
	})
}
