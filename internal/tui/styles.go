package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Title and header styles.
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))

	// Section styles.
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))

	// Status styles.
	goodStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
	critStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	labelStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))

	// Gauge characters.
	gaugeChars = []rune{'░', '▒', '▓', '█'}
)
