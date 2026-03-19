package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/73ai/openbotkit/settings"
)

type state int

const (
	stateBrowse state = iota
	stateEdit
)

type flashMsg struct{}

type model struct {
	svc      *settings.Service
	rows     []row
	expanded map[string]bool
	cursor   int
	state    state
	form     *huh.Form
	editField *settings.Field
	editStr  *string
	editBool *bool
	flash    string
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

func newModel(svc *settings.Service) model {
	expanded := make(map[string]bool)
	rows := flattenTree(svc.Tree(), expanded)
	return model{
		svc:      svc,
		rows:     rows,
		expanded: expanded,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateEdit:
		return m.updateEdit(msg)
	default:
		return m.updateBrowse(msg)
	}
}

func (m model) updateBrowse(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.viewport = viewport.New(msg.Width, msg.Height-4)
		m.viewport.SetContent(m.renderTree())
		return m, nil

	case flashMsg:
		m.flash = ""
		m.viewport.SetContent(m.renderTree())
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.viewport.SetContent(m.renderTree())
			m.ensureCursorVisible()
			return m, nil
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
			m.viewport.SetContent(m.renderTree())
			m.ensureCursorVisible()
			return m, nil
		case "enter":
			return m.handleEnter()
		}
	}
	return m, nil
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.rows) {
		return m, nil
	}
	r := m.rows[m.cursor]

	if r.node.Category != nil {
		key := r.node.Category.Key
		m.expanded[key] = !m.expanded[key]
		m.rebuildRows()
		m.viewport.SetContent(m.renderTree())
		return m, nil
	}

	if r.node.Field != nil {
		f := r.node.Field
		current := m.svc.GetValue(f)
		form, strVal, boolVal := buildForm(f, current, m.svc)
		m.state = stateEdit
		m.form = form
		m.editField = f
		m.editStr = strVal
		m.editBool = boolVal
		return m, m.form.Init()
	}

	return m, nil
}

func (m model) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.state = stateBrowse
			m.form = nil
			m.editField = nil
			m.viewport.SetContent(m.renderTree())
			return m, nil
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		var value string
		if m.editBool != nil {
			value = strconv.FormatBool(*m.editBool)
		} else if m.editStr != nil {
			value = *m.editStr
		}

		if err := m.svc.SetValue(m.editField, value); err != nil {
			m.flash = fmt.Sprintf("Error: %v", err)
		} else {
			m.flash = "Saved!"
			if m.editField.AfterSet != nil {
				if msg := m.editField.AfterSet(m.svc); msg != "" {
					m.flash = "Saved! " + msg
				}
			}
		}

		m.state = stateBrowse
		m.form = nil
		m.editField = nil
		m.rebuildRows()
		m.viewport.SetContent(m.renderTree())
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return flashMsg{}
		})
	}

	return m, cmd
}

func (m *model) rebuildRows() {
	m.rows = flattenTree(m.svc.Tree(), m.expanded)
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
}

func (m *model) ensureCursorVisible() {
	if m.cursor < m.viewport.YOffset {
		m.viewport.SetYOffset(m.cursor)
	} else if m.cursor >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(m.cursor - m.viewport.Height + 1)
	}
}

func (m model) renderTree() string {
	var b strings.Builder
	for i, r := range m.rows {
		b.WriteString(renderRow(r, m.svc, i == m.cursor))
		b.WriteString("\n")
	}
	if m.flash != "" {
		b.WriteString("\n")
		b.WriteString(flashStyle.Render("  " + m.flash))
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) View() string {
	if m.state == stateEdit && m.form != nil {
		return "\n" + m.form.View()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("  obk settings"))
	b.WriteString("\n")

	if m.ready {
		b.WriteString(m.viewport.View())
	} else {
		b.WriteString(m.renderTree())
	}

	b.WriteString(helpStyle.Render("  ↑↓ navigate  enter edit/expand  q quit"))
	b.WriteString("\n")
	return b.String()
}
