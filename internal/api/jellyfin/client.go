package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/logger"
	"github.com/codila125/musica/internal/models"
)

var _ api.Client = (*Client)(nil)

type Client struct {
	baseURL string
	userID  string
	apiKey  string
	client  *http.Client
	debug   bool
}

type jellyfinArtist struct {
	ID         string            `json:"Id"`
	Name       string            `json:"Name"`
	AlbumCount int               `json:"AlbumCount"`
	ImageTags  map[string]string `json:"ImageTags"`
}

type jellyfinAlbum struct {
	ID          string   `json:"Id"`
	Name        string   `json:"Name"`
	Artists     []string `json:"Artists,omitempty"`
	ArtistItems []struct {
		ID   string `json:"Id"`
		Name string `json:"Name"`
	} `json:"ArtistItems"`
	ProductionYear int               `json:"ProductionYear"`
	ImageTags      map[string]string `json:"ImageTags"`
}

type jellyfinTrack struct {
	ID          string   `json:"Id"`
	Name        string   `json:"Name"`
	Artists     []string `json:"Artists,omitempty"`
	ArtistItems []struct {
		ID   string `json:"Id"`
		Name string `json:"Name"`
	} `json:"ArtistItems"`
	Album        string            `json:"Album"`
	AlbumID      string            `json:"AlbumId"`
	RunTimeTicks int64             `json:"RunTimeTicks"`
	IndexNumber  int               `json:"IndexNumber"`
	ImageTags    map[string]string `json:"ImageTags"`
	MediaSources []struct {
		Container string `json:"Container"`
	} `json:"MediaSources,omitempty"`
}

type jellyfinPlaylist struct {
	ID         string            `json:"Id"`
	Name       string            `json:"Name"`
	ChildCount int               `json:"ChildCount"`
	ImageTags  map[string]string `json:"ImageTags"`
}

func New(cfg config.ServerConfig) *Client {
	return &Client{
		baseURL: cfg.URL,
		client:  &http.Client{Timeout: 30 * time.Second},
		debug:   true,
	}
}

func (c *Client) Authenticate(ctx context.Context, username, password string) error {
	return c.authenticate(ctx, username, password)
}

func (c *Client) authenticate(ctx context.Context, username, password string) error {
	payload := map[string]string{
		"Username": username,
		"Pw":       password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoints := []string{
		"/Users/Authenticate",
		"/Users/AuthenticateByName",
	}

	var lastErr error
	for _, endpoint := range endpoints {
		u := c.baseURL + endpoint

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(body))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Emby-Authorization", fmt.Sprintf(`MediaBrowser Client="musica", Version="0.1.0", DeviceId="musica-tui", Device="Terminal"`))

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("auth failed (HTTP %d): %s", resp.StatusCode, string(respBody))
			continue
		}

		var authResp struct {
			User struct {
				ID string `json:"Id"`
			} `json:"User"`
			AccessToken string `json:"AccessToken"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			lastErr = fmt.Errorf("parse auth response: %w", err)
			continue
		}

		c.userID = authResp.User.ID
		c.apiKey = authResp.AccessToken
		return nil
	}

	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("no compatible auth endpoint found")
}

func (c *Client) authHeader() http.Header {
	h := make(http.Header)
	h.Set("X-Emby-Token", c.apiKey)
	h.Set("X-Emby-Authorization", fmt.Sprintf(`MediaBrowser Client="musica", Version="0.1.0", UserId="%s"`, c.userID))
	return h
}

func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values, v interface{}) error {
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	if params != nil {
		q := u.Query()
		for k, vs := range params {
			for _, v := range vs {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}

	if c.debug {
		logger.Get().Debug("GET %s", u.String())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header = c.authHeader()

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := http.Get(c.baseURL + "/System/Info/Public")
	return err
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
		artistName := ""
		artistID := ""
		if len(t.ArtistItems) > 0 {
			artistName = t.ArtistItems[0].Name
			artistID = t.ArtistItems[0].ID
		} else if len(t.Artists) > 0 {
			artistName = t.Artists[0]
		}

		duration := int(t.RunTimeTicks / 10000000)
		format := ""
		if len(t.MediaSources) > 0 && t.MediaSources[0].Container != "" {
			format = strings.ToUpper(t.MediaSources[0].Container)
		}
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

	if c.debug {
		logger.Get().Debug("Found %d recent tracks", len(tracks))
	}

	return tracks, nil
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
		if c.debug {
			logger.Get().Debug("/Artists/AlbumArtists failed: %v, trying /Artists", err)
		}
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

	if c.debug {
		logger.Get().Debug("Found %d artists", len(artists))
	}

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
			if c.debug {
				logger.Get().Debug("/Items with ArtistIds failed: %v, trying without", err)
			}
			delete(params, "ArtistIds")
			if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if artistID != "" && len(resp.Items) == 0 {
		if c.debug {
			logger.Get().Debug("No albums for artist filter, falling back to all albums")
		}
		delete(params, "ArtistIds")
		if err := c.doRequest(ctx, "/Items", params, &resp); err != nil {
			return nil, err
		}
	}

	albums := make([]models.Album, 0, len(resp.Items))
	for _, a := range resp.Items {
		artistName := ""
		artistID := ""
		if len(a.ArtistItems) > 0 {
			artistName = a.ArtistItems[0].Name
			artistID = a.ArtistItems[0].ID
		} else if len(a.Artists) > 0 {
			artistName = a.Artists[0]
		}

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

	if c.debug {
		logger.Get().Debug("Found %d albums", len(albums))
	}

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
		artistName := ""
		artistID := ""
		if len(t.ArtistItems) > 0 {
			artistName = t.ArtistItems[0].Name
			artistID = t.ArtistItems[0].ID
		} else if len(t.Artists) > 0 {
			artistName = t.Artists[0]
		}

		duration := int(t.RunTimeTicks / 10000000)
		format := ""
		if len(t.MediaSources) > 0 && t.MediaSources[0].Container != "" {
			format = strings.ToUpper(t.MediaSources[0].Container)
		}

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

	if c.debug {
		logger.Get().Debug("Found %d tracks", len(tracks))
	}

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
		artistName := ""
		artistID := ""
		if len(t.ArtistItems) > 0 {
			artistName = t.ArtistItems[0].Name
			artistID = t.ArtistItems[0].ID
		} else if len(t.Artists) > 0 {
			artistName = t.Artists[0]
		}

		duration := int(t.RunTimeTicks / 10000000)
		format := ""
		if len(t.MediaSources) > 0 && t.MediaSources[0].Container != "" {
			format = strings.ToUpper(t.MediaSources[0].Container)
		}

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
			ID          string   `json:"Id"`
			ItemID      string   `json:"ItemId"`
			Name        string   `json:"Name"`
			Type        string   `json:"Type"`
			Album       string   `json:"Album"`
			AlbumID     string   `json:"AlbumId"`
			Artists     []string `json:"Artists,omitempty"`
			ArtistItems []struct {
				ID   string `json:"Id"`
				Name string `json:"Name"`
			} `json:"ArtistItems"`
			RunTimeTicks int64 `json:"RunTimeTicks"`
			IndexNumber  int   `json:"IndexNumber"`
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
		artistName := ""
		artistID := ""
		if len(h.ArtistItems) > 0 {
			artistName = h.ArtistItems[0].Name
			artistID = h.ArtistItems[0].ID
		} else if len(h.Artists) > 0 {
			artistName = h.Artists[0]
		}

		switch h.Type {
		case "Audio":
			duration := int(h.RunTimeTicks / 10000000)
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

func (c *Client) StreamTrack(ctx context.Context, trackID string) (io.ReadCloser, error) {
	u := c.baseURL + fmt.Sprintf("/Items/%s/Download?api_key=%s", trackID, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header = c.authHeader()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c *Client) GetStreamURL(trackID string) string {
	return c.getStreamURL(trackID)
}

func (c *Client) getStreamURL(trackID string) string {
	return c.baseURL + fmt.Sprintf("/Items/%s/Download?api_key=%s", trackID, c.apiKey)
}

func (c *Client) GetCoverURL(albumID string) string {
	return c.getCoverURL(albumID)
}

func (c *Client) getCoverURL(itemID string) string {
	if itemID == "" {
		return ""
	}
	return fmt.Sprintf("%s/Items/%s/Images/Primary?api_key=%s", c.baseURL, itemID, c.apiKey)
}
