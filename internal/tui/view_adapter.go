package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/tui/views"
)

type viewAdapter struct {
	browse     views.BrowseModel
	search     views.SearchModel
	library    views.LibraryModel
	playlists  views.PlaylistsModel
	queue      views.QueueModel
	nowPlaying views.NowPlayingModel
}

func newViewAdapter(client api.Client, playback views.PlaybackService) viewAdapter {
	return viewAdapter{
		browse:     views.NewBrowseModel(client, playback),
		search:     views.NewSearchModel(client, playback),
		library:    views.NewLibraryModel(client, playback),
		playlists:  views.NewPlaylistsModel(client, playback),
		queue:      views.NewQueueModel(playback),
		nowPlaying: views.NewNowPlayingModel(client, playback),
	}
}

func (v *viewAdapter) Init() tea.Cmd {
	return tea.Batch(v.browse.Init(), v.search.Init(), v.library.Init(), v.playlists.Init(), v.queue.Init(), v.nowPlaying.Init())
}

func (v *viewAdapter) Resize(msg tea.WindowSizeMsg) {
	v.browse, _ = v.browse.Update(msg)
	v.search, _ = v.search.Update(msg)
	v.library, _ = v.library.Update(msg)
	v.playlists, _ = v.playlists.Update(msg)
	v.queue, _ = v.queue.Update(msg)
	v.nowPlaying, _ = v.nowPlaying.Update(msg)
}

func (v *viewAdapter) CancelInFlight() {
	v.browse, _ = v.browse.Update(views.CancelInFlightCmd())
	v.search, _ = v.search.Update(views.CancelInFlightCmd())
	v.library, _ = v.library.Update(views.CancelInFlightCmd())
	v.playlists, _ = v.playlists.Update(views.CancelInFlightCmd())
}

func (v *viewAdapter) UpdateAll(msg tea.Msg) tea.Cmd {
	var cmdBrowse, cmdSearch, cmdLibrary, cmdPlaylists, cmdQueue, cmdNowPlaying tea.Cmd
	v.browse, cmdBrowse = v.browse.Update(msg)
	v.search, cmdSearch = v.search.Update(msg)
	v.library, cmdLibrary = v.library.Update(msg)
	v.playlists, cmdPlaylists = v.playlists.Update(msg)
	v.queue, cmdQueue = v.queue.Update(msg)
	v.nowPlaying, cmdNowPlaying = v.nowPlaying.Update(msg)
	return tea.Batch(cmdBrowse, cmdSearch, cmdLibrary, cmdPlaylists, cmdQueue, cmdNowPlaying)
}

func (v *viewAdapter) UpdateActive(tab Tab, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch tab {
	case TabBrowse:
		v.browse, cmd = v.browse.Update(msg)
	case TabSearch:
		v.search, cmd = v.search.Update(msg)
	case TabLibrary:
		v.library, cmd = v.library.Update(msg)
	case TabPlaylists:
		v.playlists, cmd = v.playlists.Update(msg)
	case TabQueue:
		v.queue, cmd = v.queue.Update(msg)
	case TabNowPlaying:
		v.nowPlaying, cmd = v.nowPlaying.Update(msg)
	}
	return cmd
}

func (v *viewAdapter) View(tab Tab) string {
	switch tab {
	case TabBrowse:
		return v.browse.View()
	case TabSearch:
		return v.search.View()
	case TabLibrary:
		return v.library.View()
	case TabPlaylists:
		return v.playlists.View()
	case TabQueue:
		return v.queue.View()
	case TabNowPlaying:
		return v.nowPlaying.View()
	default:
		return ""
	}
}

func (v *viewAdapter) SearchIsInInputMode() bool {
	return v.search.InInputMode()
}
