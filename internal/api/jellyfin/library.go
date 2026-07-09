package jellyfin

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/codila125/musica/internal/logger"
	"github.com/codila125/musica/internal/models"
)

type jellyfinArtist struct {
	ID         string            `json:"Id"`
	Name       string            `json:"Name"`
	AlbumCount int               `json:"AlbumCount"`
	ImageTags  map[string]string `json:"ImageTags"`
}

type jellyfinAlbum struct {
	ID             string              `json:"Id"`
	Name           string              `json:"Name"`
	Artists        []string            `json:"Artists,omitempty"`
	ArtistItems    []jellyfinArtistRef `json:"ArtistItems"`
	ProductionYear int                 `json:"ProductionYear"`
	ImageTags      map[string]string   `json:"ImageTags"`
}

type jellyfinTrack struct {
	ID           string                `json:"Id"`
	Name         string                `json:"Name"`
	Artists      []string              `json:"Artists,omitempty"`
	ArtistItems  []jellyfinArtistRef   `json:"ArtistItems"`
	Album        string                `json:"Album"`
	AlbumID      string                `json:"AlbumId"`
	RunTimeTicks int64                 `json:"RunTimeTicks"`
	IndexNumber  int                   `json:"IndexNumber"`
	ImageTags    map[string]string     `json:"ImageTags"`
	MediaSources []jellyfinMediaSource `json:"MediaSources,omitempty"`
}

type jellyfinMediaSource struct {
	Container string `json:"Container"`
}

type jellyfinPlaylist struct {
	ID         string            `json:"Id"`
	Name       string            `json:"Name"`
	ChildCount int               `json:"ChildCount"`
	ImageTags  map[string]string `json:"ImageTags"`
}

// jellyfinArtistRef matches the ArtistItems element shape shared by
// jellyfinTrack, jellyfinAlbum, and the inline search-hint struct.
type jellyfinArtistRef struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

func resolveArtist(items []jellyfinArtistRef, names []string) (name, id string) {
	if len(items) > 0 {
		return items[0].Name, items[0].ID
	}
	if len(names) > 0 {
		return names[0], ""
	}
	return "", ""
}

func ticksToSeconds(ticks int64) int {
	return int(ticks / 10000000)
}

func mediaFormat(sources []jellyfinMediaSource) string {
	if len(sources) > 0 && sources[0].Container != "" {
		return strings.ToUpper(sources[0].Container)
	}
	return ""
}

func (c *Client) GetRecentTracks(ctx context.Context, limit int) ([]models.Track, error) {
	if limit <= 0 {
		limit = 50
	}

	var resp struct {
		Items []jellyfinTrack `json:"Items"`
	}

	params := url.Values{
		"UserId":           {c.userID},
		"IncludeItemTypes": {"Audio"},
		"Recursive":        {"true"},
		"SortBy":           {"DateCreated,SortName"},
		"SortOrder":        {"Descending"},
		"Limit":            {fmt.Sprintf("%d", limit)},
	}

	if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
		return nil, err
	}

	tracks := make([]models.Track, 0, len(resp.Items))
	for _, t := range resp.Items {
		artistName, artistID := resolveArtist(t.ArtistItems, t.Artists)
		duration := ticksToSeconds(t.RunTimeTicks)
		format := mediaFormat(t.MediaSources)
		tracks = append(tracks, models.Track{
			ID:        t.ID,
			Title:     t.Name,
			Artist:    artistName,
			ArtistID:  artistID,
			Album:     t.Album,
			AlbumID:   t.AlbumID,
			Duration:  duration,
			TrackNum:  t.IndexNumber,
			StreamURL: c.getStreamURL(t.ID),
			CoverURL:  c.getCoverURL(t.AlbumID),
			Format:    format,
		})
	}

	logger.Get().Debug("Found %d recent tracks", len(tracks))

	return tracks, nil
}

func (c *Client) GetRecentTracksCount(ctx context.Context) (int, error) {
	var resp struct {
		TotalRecordCount int `json:"TotalRecordCount"`
	}

	params := url.Values{
		"UserId":           {c.userID},
		"IncludeItemTypes": {"Audio"},
		"Recursive":        {"true"},
		"SortBy":           {"DateCreated,SortName"},
		"SortOrder":        {"Descending"},
		"Limit":            {"1"},
		"Fields":           {""},
	}

	if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
		return 0, err
	}

	return resp.TotalRecordCount, nil
}

func (c *Client) GetArtists(ctx context.Context) ([]models.Artist, error) {
	var resp struct {
		Items []jellyfinArtist `json:"Items"`
	}

	params := url.Values{
		"UserId":    {c.userID},
		"Recursive": {"true"},
		"SortBy":    {"SortName"},
	}

	if err := c.doRequest(ctx, "/Artists/AlbumArtists", params, &resp); err != nil {
		logger.Get().Debug("/Artists/AlbumArtists failed: %v, trying /Artists", err)
		if err := c.doRequest(ctx, "/Artists", params, &resp); err != nil {
			return nil, err
		}
	}

	artists := make([]models.Artist, 0, len(resp.Items))
	for _, a := range resp.Items {
		coverURL := ""
		if a.ImageTags != nil {
			if _, ok := a.ImageTags["Primary"]; ok {
				coverURL = c.getCoverURL(a.ID)
			}
		}
		artists = append(artists, models.Artist{
			ID:         a.ID,
			Name:       a.Name,
			AlbumCount: a.AlbumCount,
			CoverURL:   coverURL,
		})
	}

	logger.Get().Debug("Found %d artists", len(artists))

	return artists, nil
}

func (c *Client) GetAlbums(ctx context.Context, artistID string) ([]models.Album, error) {
	var resp struct {
		Items []jellyfinAlbum `json:"Items"`
	}

	params := url.Values{
		"UserId":           {c.userID},
		"IncludeItemTypes": {"MusicAlbum"},
		"Recursive":        {"true"},
		"SortBy":           {"SortName"},
		"Limit":            {"500"},
	}

	if artistID != "" {
		params.Set("ArtistIds", artistID)
	}

	if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
		if artistID != "" {
			logger.Get().Debug("/Items with ArtistIds failed: %v, trying without", err)
			delete(params, "ArtistIds")
			if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if artistID != "" && len(resp.Items) == 0 {
		logger.Get().Debug("No albums for artist filter, falling back to all albums")
		delete(params, "ArtistIds")
		if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
			return nil, err
		}
	}

	albums := make([]models.Album, 0, len(resp.Items))
	for _, a := range resp.Items {
		artistName, artistID := resolveArtist(a.ArtistItems, a.Artists)

		coverURL := ""
		if a.ImageTags != nil {
			if _, ok := a.ImageTags["Primary"]; ok {
				coverURL = c.getCoverURL(a.ID)
			}
		}

		albums = append(albums, models.Album{
			ID:       a.ID,
			Name:     a.Name,
			Artist:   artistName,
			ArtistID: artistID,
			Year:     a.ProductionYear,
			CoverURL: coverURL,
		})
	}

	logger.Get().Debug("Found %d albums", len(albums))

	return albums, nil
}

func (c *Client) GetTracks(ctx context.Context, albumID string) ([]models.Track, error) {
	var resp struct {
		Items []jellyfinTrack `json:"Items"`
	}

	params := url.Values{
		"UserId":           {c.userID},
		"ParentId":         {albumID},
		"Recursive":        {"true"},
		"IncludeItemTypes": {"Audio"},
		"SortBy":           {"IndexNumber", "SortName"},
	}

	if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
		return nil, err
	}

	tracks := make([]models.Track, 0, len(resp.Items))
	for _, t := range resp.Items {
		artistName, artistID := resolveArtist(t.ArtistItems, t.Artists)
		duration := ticksToSeconds(t.RunTimeTicks)
		format := mediaFormat(t.MediaSources)

		tracks = append(tracks, models.Track{
			ID:        t.ID,
			Title:     t.Name,
			Artist:    artistName,
			ArtistID:  artistID,
			Album:     t.Album,
			AlbumID:   t.AlbumID,
			Duration:  duration,
			TrackNum:  t.IndexNumber,
			StreamURL: c.getStreamURL(t.ID),
			CoverURL:  c.getCoverURL(t.AlbumID),
			Format:    format,
		})
	}

	logger.Get().Debug("Found %d tracks", len(tracks))

	return tracks, nil
}

func (c *Client) GetPlaylists(ctx context.Context) ([]models.Playlist, error) {
	var resp struct {
		Items []jellyfinPlaylist `json:"Items"`
	}

	params := url.Values{
		"UserId":           {c.userID},
		"Recursive":        {"true"},
		"IncludeItemTypes": {"Playlist"},
	}

	if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
		return nil, err
	}

	playlists := make([]models.Playlist, 0, len(resp.Items))
	for _, p := range resp.Items {
		coverURL := ""
		if p.ImageTags != nil {
			if _, ok := p.ImageTags["Primary"]; ok {
				coverURL = c.getCoverURL(p.ID)
			}
		}

		playlists = append(playlists, models.Playlist{
			ID:         p.ID,
			Name:       p.Name,
			TrackCount: p.ChildCount,
			CoverURL:   coverURL,
		})
	}

	return playlists, nil
}

func (c *Client) GetPlaylistTracks(ctx context.Context, playlistID string) ([]models.Track, error) {
	var resp struct {
		Items []jellyfinTrack `json:"Items"`
	}

	params := url.Values{
		"UserId":    {c.userID},
		"ParentId":  {playlistID},
		"Recursive": {"true"},
		"SortBy":    {"SortName"},
	}

	if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
		return nil, err
	}

	tracks := make([]models.Track, 0, len(resp.Items))
	for _, t := range resp.Items {
		artistName, artistID := resolveArtist(t.ArtistItems, t.Artists)
		duration := ticksToSeconds(t.RunTimeTicks)
		format := mediaFormat(t.MediaSources)

		tracks = append(tracks, models.Track{
			ID:        t.ID,
			Title:     t.Name,
			Artist:    artistName,
			ArtistID:  artistID,
			Album:     t.Album,
			AlbumID:   t.AlbumID,
			Duration:  duration,
			TrackNum:  t.IndexNumber,
			StreamURL: c.getStreamURL(t.ID),
			CoverURL:  c.getCoverURL(t.AlbumID),
			Format:    format,
		})
	}

	return tracks, nil
}

func (c *Client) Search(ctx context.Context, query string) (models.SearchResult, error) {
	var resp struct {
		SearchHints []struct {
			ID           string              `json:"Id"`
			ItemID       string              `json:"ItemId"`
			Name         string              `json:"Name"`
			Type         string              `json:"Type"`
			Album        string              `json:"Album"`
			AlbumID      string              `json:"AlbumId"`
			Artists      []string            `json:"Artists,omitempty"`
			ArtistItems  []jellyfinArtistRef `json:"ArtistItems"`
			RunTimeTicks int64               `json:"RunTimeTicks"`
			IndexNumber  int                 `json:"IndexNumber"`
		} `json:"SearchHints"`
	}

	params := url.Values{
		"UserId":     {c.userID},
		"SearchTerm": {query},
		"Limit":      {"50"},
	}

	if err := c.doRequest(ctx, "/Search/Hints", params, &resp); err != nil {
		return models.SearchResult{}, err
	}

	result := models.SearchResult{}

	for _, h := range resp.SearchHints {
		id := h.ID
		if id == "" {
			id = h.ItemID
		}
		artistName, artistID := resolveArtist(h.ArtistItems, h.Artists)

		switch h.Type {
		case "Audio":
			duration := ticksToSeconds(h.RunTimeTicks)
			result.Tracks = append(result.Tracks, models.Track{
				ID:        id,
				Title:     h.Name,
				Artist:    artistName,
				ArtistID:  artistID,
				Album:     h.Album,
				AlbumID:   h.AlbumID,
				Duration:  duration,
				TrackNum:  h.IndexNumber,
				StreamURL: c.getStreamURL(id),
				CoverURL:  c.getCoverURL(h.AlbumID),
			})
		case "MusicAlbum":
			result.Albums = append(result.Albums, models.Album{
				ID:       id,
				Name:     h.Name,
				Artist:   artistName,
				CoverURL: c.getCoverURL(id),
			})
		case "MusicArtist":
			result.Artists = append(result.Artists, models.Artist{
				ID:   id,
				Name: h.Name,
			})
		}
	}

	return result, nil
}
