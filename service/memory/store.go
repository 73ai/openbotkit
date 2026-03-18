package memory

import (
	"fmt"
	"time"

	"github.com/73ai/openbotkit/store"
)

func Add(db *store.DB, content string, category Category, source, sourceRef string) (int64, error) {
	res, err := db.Exec(
		db.Rebind("INSERT INTO memories (content, category, source, source_ref) VALUES (?, ?, ?, ?)"),
		content, string(category), source, sourceRef,
	)
	if err != nil {
		return 0, fmt.Errorf("insert memory: %w", err)
	}
	return res.LastInsertId()
}

func Update(db *store.DB, id int64, content string) error {
	_, err := db.Exec(
		db.Rebind("UPDATE memories SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"),
		content, id,
	)
	if err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	return nil
}

func Delete(db *store.DB, id int64) error {
	_, err := db.Exec(
		db.Rebind("DELETE FROM memories WHERE id = ?"),
		id,
	)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

func Get(db *store.DB, id int64) (*Memory, error) {
	var m Memory
	var cat, src string
	var srcRef *string
	var createdAt, updatedAt string
	err := db.QueryRow(
		db.Rebind("SELECT id, content, category, source, source_ref, created_at, updated_at FROM memories WHERE id = ?"),
		id,
	).Scan(&m.ID, &m.Content, &cat, &src, &srcRef, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}
	m.Category = Category(cat)
	m.Source = src
	if srcRef != nil {
		m.SourceRef = *srcRef
	}
	m.CreatedAt = parseTime(createdAt)
	m.UpdatedAt = parseTime(updatedAt)
	return &m, nil
}

func List(db *store.DB) ([]Memory, error) {
	return queryMemories(db, "SELECT id, content, category, source, source_ref, created_at, updated_at FROM memories ORDER BY category, created_at")
}

func ListByCategory(db *store.DB, category Category) ([]Memory, error) {
	return queryMemories(db,
		db.Rebind("SELECT id, content, category, source, source_ref, created_at, updated_at FROM memories WHERE category = ? ORDER BY created_at"),
		string(category),
	)
}

func Search(db *store.DB, query string) ([]Memory, error) {
	pattern := "%" + query + "%"
	return queryMemories(db,
		db.Rebind("SELECT id, content, category, source, source_ref, created_at, updated_at FROM memories WHERE content LIKE ? ORDER BY category, created_at"),
		pattern,
	)
}

func Count(db *store.DB) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM memories").Scan(&count)
	return count, err
}

func queryMemories(db *store.DB, query string, args ...any) ([]Memory, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query memories: %w", err)
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		var cat, src string
		var srcRef *string
		var createdAt, updatedAt string
		if err := rows.Scan(&m.ID, &m.Content, &cat, &src, &srcRef, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		m.Category = Category(cat)
		m.Source = src
		if srcRef != nil {
			m.SourceRef = *srcRef
		}
		m.CreatedAt = parseTime(createdAt)
		m.UpdatedAt = parseTime(updatedAt)
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func parseTime(s string) time.Time {
	for _, f := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	} {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
