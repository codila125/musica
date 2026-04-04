package views

import "github.com/charmbracelet/lipgloss"

var (
	retroPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("99")).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("0")).
			Padding(1, 2)

	retroTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)

	retroSubtleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	retroSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).
				Bold(true)

	retroCurrentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51")).
				Bold(true)

	retroErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	retroLoadingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("221"))
)

func retroPanelForWidth(width int) lipgloss.Style {
	panelStyle := retroPanelStyle.Copy()
	if width <= 0 {
		return panelStyle
	}

	panelWidth := width - 8
	if panelWidth < 24 {
		panelWidth = 24
	}

	return panelStyle.Width(panelWidth).MaxWidth(panelWidth)
}
