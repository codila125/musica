package views

import "github.com/charmbracelet/lipgloss"

// Retro cassette player color palette
var (
	// Main colors
	colorPurpleBorder = lipgloss.Color("93")
	colorLightText    = lipgloss.Color("230")
	colorDimText      = lipgloss.Color("244")
	colorYellowTitle  = lipgloss.Color("226")
	colorGreenSelect  = lipgloss.Color("46")
	colorCyanPlaying  = lipgloss.Color("51")
	colorRedBlink     = lipgloss.Color("196")
	colorRedDim       = lipgloss.Color("88")
	colorAmber        = lipgloss.Color("214")
	colorBrown        = lipgloss.Color("130")

	retroTitleStyle = lipgloss.NewStyle().
			Foreground(colorYellowTitle).
			Bold(true)

	retroSubtleStyle = lipgloss.NewStyle().
				Foreground(colorDimText)

	retroSelectedStyle = lipgloss.NewStyle().
				Foreground(colorGreenSelect).
				Bold(true)

	retroCurrentStyle = lipgloss.NewStyle().
				Foreground(colorCyanPlaying).
				Bold(true)

	retroErrorStyle = lipgloss.NewStyle().
			Foreground(colorRedBlink).
			Bold(true)

	retroLoadingStyle = lipgloss.NewStyle().
				Foreground(colorAmber)

	retroCassetteStyle = lipgloss.NewStyle().
				Foreground(colorBrown)

	retroColumnHeaderStyle = lipgloss.NewStyle().
				Foreground(colorAmber).
				Bold(true)
)
