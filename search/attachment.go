package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SaveAttachments saves all attachments from an email to disk under baseDir/serviceName/.
// It updates each attachment's SavedPath field.
func SaveAttachments(email *Email, baseDir string, serviceName string) error {
	if len(email.Attachments) == 0 {
		return nil
	}

	// Sanitize service name for directory
	safeName := strings.ReplaceAll(serviceName, " ", "_")
	dir := filepath.Join(baseDir, safeName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create attachment dir %s: %w", dir, err)
	}

	for i := range email.Attachments {
		att := &email.Attachments[i]
		if len(att.Data) == 0 {
			continue
		}

		// Sanitize filename: replace path separators and keep only the base name
		safeFilename := filepath.Base(strings.ReplaceAll(att.Filename, "/", "_"))
		safeFilename = strings.ReplaceAll(safeFilename, "\\", "_")

		savePath := filepath.Join(dir, safeFilename)

		// Avoid overwriting: append message ID if file exists
		if _, err := os.Stat(savePath); err == nil {
			ext := filepath.Ext(safeFilename)
			base := strings.TrimSuffix(safeFilename, ext)
			savePath = filepath.Join(dir, base+"_"+email.MessageID+ext)
		}

		if err := os.WriteFile(savePath, att.Data, 0644); err != nil {
			return fmt.Errorf("write attachment %s: %w", att.Filename, err)
		}
		att.SavedPath = savePath
	}
	return nil
}
