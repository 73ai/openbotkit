package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			MarginBottom(1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	categoryStyle = lipgloss.NewStyle().
			Bold(true)

	fieldLabelStyle = lipgloss.NewStyle()

	fieldValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	passwordConfiguredStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("2"))

	passwordNotConfiguredStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("1"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1)

	flashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1"))
)
