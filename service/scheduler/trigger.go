package scheduler

import (
	"fmt"
	"strings"

	"github.com/73ai/openbotkit/store"
)

var triggerTemplates = map[string]string{
	"gmail":      `SELECT id, subject, from_addr, date FROM emails WHERE id > ? AND (%s) ORDER BY id LIMIT 50`,
	"whatsapp":   `SELECT id, message_id, sender_name, text, timestamp FROM whatsapp_messages WHERE id > ? AND (%s) ORDER BY id LIMIT 50`,
	"imessage":   `SELECT id, guid, sender_id, text, date_utc FROM imessage_messages WHERE id > ? AND (%s) ORDER BY id LIMIT 50`,
	"applenotes": `SELECT id, title, folder, modified_at FROM applenotes_notes WHERE id > ? AND (%s) ORDER BY id LIMIT 50`,
}

func ValidateTriggerSource(source string) error {
	if _, ok := triggerTemplates[source]; !ok {
		return fmt.Errorf("unknown trigger source %q; supported: gmail, whatsapp, imessage, applenotes", source)
	}
	return nil
}

var dangerousKeywords = []string{";", "DROP", "DELETE", "INSERT", "UPDATE", "CREATE", "ALTER", "--"}

func ValidateTriggerQuery(query string) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("trigger query must not be empty")
	}
	upper := strings.ToUpper(query)
	for _, kw := range dangerousKeywords {
		if strings.Contains(upper, kw) {
			return fmt.Errorf("trigger query must not contain %q", kw)
		}
	}
	if strings.Count(query, "(") != strings.Count(query, ")") {
		return fmt.Errorf("trigger query has unbalanced parentheses")
	}
	return nil
}

func BuildTriggerQuery(source, whereClause string, lastTriggerID int64) (string, []any, error) {
	tmpl, ok := triggerTemplates[source]
	if !ok {
		return "", nil, fmt.Errorf("unknown trigger source %q", source)
	}
	q := fmt.Sprintf(tmpl, whereClause)
	return q, []any{lastTriggerID}, nil
}

type TriggerMatch struct {
	MaxID int64
	Rows  []map[string]string
}

func CheckTrigger(db *store.DB, source, whereClause string, lastTriggerID int64) (*TriggerMatch, error) {
	query, args, err := BuildTriggerQuery(source, whereClause, lastTriggerID)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("trigger query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("trigger columns: %w", err)
	}

	var match TriggerMatch
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("trigger scan: %w", err)
		}
		row := make(map[string]string, len(cols))
		for i, col := range cols {
			row[col] = fmt.Sprintf("%v", vals[i])
		}
		if id, ok := row["id"]; ok {
			var n int64
			fmt.Sscanf(id, "%d", &n)
			if n > match.MaxID {
				match.MaxID = n
			}
		}
		match.Rows = append(match.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("trigger rows: %w", err)
	}

	if len(match.Rows) == 0 {
		return nil, nil
	}
	return &match, nil
}
