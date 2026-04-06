package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/tui/views"
)

type viewAdapter struct {
	browse views.BrowseModel
	search views.SearchModel
	queue  views.QueueModel
}

func newViewAdapter(client api.Client, playback views.PlaybackService) viewAdapter {
	return viewAdapter{
		browse: views.NewBrowseModel(client, playback),
		search: views.NewSearchModel(client, playback),
		queue:  views.NewQueueModel(playback),
	}
}

func (v *viewAdapter) Init() tea.Cmd {
	return tea.Batch(v.browse.Init(), v.search.Init(), v.queue.Init())
}

func (v *viewAdapter) Resize(msg tea.WindowSizeMsg) {
	v.browse, _ = v.browse.Update(msg)
	v.search, _ = v.search.Update(msg)
	v.queue, _ = v.queue.Update(msg)
}

func (v *viewAdapter) CancelInFlight() {
	v.browse, _ = v.browse.Update(views.CancelInFlightCmd())
	v.search, _ = v.search.Update(views.CancelInFlightCmd())
}

func (v *viewAdapter) UpdateAll(msg tea.Msg) tea.Cmd {
	var cmdBrowse, cmdSearch, cmdQueue tea.Cmd
	v.browse, cmdBrowse = v.browse.Update(msg)
	v.search, cmdSearch = v.search.Update(msg)
	v.queue, cmdQueue = v.queue.Update(msg)
	return tea.Batch(cmdBrowse, cmdSearch, cmdQueue)
}

func (v *viewAdapter) UpdateActive(tab Tab, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch tab {
	case TabBrowse:
		v.browse, cmd = v.browse.Update(msg)
	case TabSearch:
		v.search, cmd = v.search.Update(msg)
	case TabQueue:
		v.queue, cmd = v.queue.Update(msg)
	}
	return cmd
}

func (v *viewAdapter) View(tab Tab) string {
	switch tab {
	case TabBrowse:
		return v.browse.View()
	case TabSearch:
		return v.search.View()
	case TabQueue:
		return v.queue.View()
	default:
		return ""
	}
}

func (v *viewAdapter) SearchIsInInputMode() bool {
	return v.search.InInputMode()
}
