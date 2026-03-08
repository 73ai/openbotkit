package skills

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// IndexEntry represents a single skill in the index.
type IndexEntry struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Index holds the skill index.
type Index struct {
	Skills []IndexEntry `yaml:"skills"`
}

// GenerateIndex reads all SKILL.md files in the skills directory,
// parses their YAML frontmatter, and writes an index.yaml file.
func GenerateIndex() error {
	dir := SkillsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read skills dir: %w", err)
	}

	var index Index
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
		ie, err := parseSkillFrontmatter(skillPath)
		if err != nil {
			// Skip skills with bad frontmatter.
			continue
		}
		index.Skills = append(index.Skills, ie)
	}

	return SaveIndex(&index)
}

// LoadIndex reads the skill index from disk.
func LoadIndex() (*Index, error) {
	data, err := os.ReadFile(IndexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{}, nil
		}
		return nil, fmt.Errorf("read index: %w", err)
	}
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	return &idx, nil
}

// SaveIndex writes the skill index to disk.
func SaveIndex(idx *Index) error {
	if err := os.MkdirAll(SkillsDir(), 0700); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}
	data, err := yaml.Marshal(idx)
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	return os.WriteFile(IndexPath(), data, 0600)
}

// IndexPath returns the path to the skill index file.
func IndexPath() string {
	return filepath.Join(SkillsDir(), "index.yaml")
}

// parseSkillFrontmatter extracts name and description from SKILL.md YAML frontmatter.
func parseSkillFrontmatter(path string) (IndexEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return IndexEntry{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	var frontmatterLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			// End of frontmatter.
			break
		}
		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		}
	}

	if len(frontmatterLines) == 0 {
		return IndexEntry{}, fmt.Errorf("no frontmatter found")
	}

	var entry IndexEntry
	if err := yaml.Unmarshal([]byte(strings.Join(frontmatterLines, "\n")), &entry); err != nil {
		return IndexEntry{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	if entry.Name == "" {
		return IndexEntry{}, fmt.Errorf("missing name in frontmatter")
	}

	return entry, nil
}
