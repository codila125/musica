package views

import tea "github.com/charmbracelet/bubbletea"

type cancelInFlightMsg struct{}

func CancelInFlightCmd() tea.Msg {
	return cancelInFlightMsg{}
}
