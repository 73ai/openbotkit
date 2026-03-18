package memory

import (
	"fmt"
	"strings"
)

var categoryLabels = map[Category]string{
	CategoryIdentity:     "Identity",
	CategoryPreference:   "Preferences",
	CategoryRelationship: "Relationships",
	CategoryProject:      "Projects & Context",
}

var categoryOrder = []Category{
	CategoryIdentity,
	CategoryPreference,
	CategoryRelationship,
	CategoryProject,
}

func FormatForPrompt(memories []Memory) string {
	if len(memories) == 0 {
		return ""
	}

	grouped := make(map[Category][]string)
	for _, m := range memories {
		grouped[m.Category] = append(grouped[m.Category], m.Content)
	}

	var sb strings.Builder
	sb.WriteString("## About the user\n")

	for _, cat := range categoryOrder {
		items := grouped[cat]
		label := categoryLabels[cat]
		sb.WriteString(fmt.Sprintf("\n### %s\n", label))
		if len(items) == 0 {
			sb.WriteString("- (none)\n")
			continue
		}
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("- %s\n", item))
		}
	}

	return sb.String()
}
