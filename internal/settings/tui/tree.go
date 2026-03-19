package tui

import (
	"fmt"
	"strings"

	"github.com/73ai/openbotkit/settings"
)

type row struct {
	node     settings.Node
	depth    int
	expanded bool
}

func flattenTree(nodes []settings.Node, expanded map[string]bool) []row {
	var rows []row
	flattenNodes(nodes, 0, expanded, &rows)
	return rows
}

func flattenNodes(nodes []settings.Node, depth int, expanded map[string]bool, rows *[]row) {
	for _, n := range nodes {
		if n.Category != nil {
			isExpanded := expanded[n.Category.Key]
			*rows = append(*rows, row{node: n, depth: depth, expanded: isExpanded})
			if isExpanded {
				flattenNodes(n.Category.Children, depth+1, expanded, rows)
			}
		} else if n.Field != nil {
			*rows = append(*rows, row{node: n, depth: depth})
		}
	}
}

func renderRow(r row, svc *settings.Service, selected bool) string {
	indent := strings.Repeat("  ", r.depth+1)
	cursor := "  "
	if selected {
		cursor = cursorStyle.Render("> ")
	}

	if r.node.Category != nil {
		arrow := ">"
		if r.expanded {
			arrow = "v"
		}
		label := categoryStyle.Render(r.node.Category.Label)
		return fmt.Sprintf("%s%s %s", cursor, indent, arrow+" "+label)
	}

	f := r.node.Field
	label := fieldLabelStyle.Render(f.Label)
	value := svc.GetValue(f)

	var valueStr string
	switch {
	case f.Type == settings.TypePassword && strings.Contains(value, "not configured"):
		valueStr = passwordNotConfiguredStyle.Render(value)
	case f.Type == settings.TypePassword:
		valueStr = passwordConfiguredStyle.Render(value)
	default:
		valueStr = fieldValueStyle.Render(value)
	}

	return fmt.Sprintf("%s%s %s: %s", cursor, indent, label, valueStr)
}
