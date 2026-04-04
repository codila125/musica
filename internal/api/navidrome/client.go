package navidrome

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/models"
)

var _ api.Client = (*Client)(nil)

type Client struct {
	baseURL string
	user    string
	token   string
	salt    string
	client  *http.Client
}

type subsonicResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`

	Artists *struct {
		Index []struct {
			Name   string `json:"name"`
			Artist []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				AlbumCount int    `json:"albumCount"`
				CoverArt   string `json:"coverArt"`
			} `json:"artist"`
		} `json:"index"`
	} `json:"artists"`

	Artist *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Album []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Artist    string `json:"artist"`
			ArtistID  string `json:"artistId"`
			Year      int    `json:"year"`
			CoverArt  string `json:"coverArt"`
			SongCount int    `json:"songCount"`
		} `json:"album"`
	} `json:"artist"`

	Album *struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Artist   string `json:"artist"`
		ArtistID string `json:"artistId"`
		Year     int    `json:"year"`
		CoverArt string `json:"coverArt"`
		Song     []struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Artist   string `json:"artist"`
			ArtistID string `json:"artistId"`
			Album    string `json:"album"`
			AlbumID  string `json:"albumId"`
			Duration int    `json:"duration"`
			Track    int    `json:"track"`
			CoverArt string `json:"coverArt"`
		} `json:"song"`
	} `json:"album"`

	AlbumList2 *struct {
		Album []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Artist   string `json:"artist"`
			ArtistID string `json:"artistId"`
			Year     int    `json:"year"`
			CoverArt string `json:"coverArt"`
		} `json:"album"`
	} `json:"albumList2"`

	Playlists *struct {
		Playlist []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			SongCount int    `json:"songCount"`
			CoverArt  string `json:"coverArt"`
		} `json:"playlist"`
	} `json:"playlists"`

	Playlist *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Entry []struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Artist   string `json:"artist"`
			ArtistID string `json:"artistId"`
			Album    string `json:"album"`
			AlbumID  string `json:"albumId"`
			Duration int    `json:"duration"`
			Track    int    `json:"track"`
			CoverArt string `json:"coverArt"`
		} `json:"entry"`
	} `json:"playlist"`

	SearchResult3 *struct {
		Artist []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			AlbumCount int    `json:"albumCount"`
			CoverArt   string `json:"coverArt"`
		} `json:"artist"`
		Album []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Artist   string `json:"artist"`
			ArtistID string `json:"artistId"`
			Year     int    `json:"year"`
			CoverArt string `json:"coverArt"`
		} `json:"album"`
		Song []struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Artist   string `json:"artist"`
			ArtistID string `json:"artistId"`
			Album    string `json:"album"`
			AlbumID  string `json:"albumId"`
			Duration int    `json:"duration"`
			Track    int    `json:"track"`
			CoverArt string `json:"coverArt"`
		} `json:"song"`
	} `json:"searchResult3"`
}

func New(cfg config.ServerConfig) *Client {
	salt := fmt.Sprintf("%d", time.Now().UnixNano())
	token := fmt.Sprintf("%x", md5.Sum([]byte(cfg.Password+salt)))

	return &Client{
		baseURL: cfg.URL,
		user:    cfg.Username,
		token:   token,
		salt:    salt,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) authParams() url.Values {
	return url.Values{
		"u": {c.user},
		"t": {c.token},
		"s": {c.salt},
		"v": {"1.16.1"},
		"c": {"musica"},
		"f": {"json"},
	}
}

func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values, v interface{}) error {
	u, err := url.Parse(c.baseURL + "/rest/" + endpoint)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	for k, vs := range params {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var wrapper struct {
		SubsonicResponse json.RawMessage `json:"subsonic-response"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return fmt.Errorf("unmarshal wrapper: %w", err)
	}

	return json.Unmarshal(wrapper.SubsonicResponse, v)
}

func (c *Client) Ping(ctx context.Context) error {
	var resp subsonicResponse
	params := c.authParams()
	return c.doRequest(ctx, "ping", params, &resp)
}

func (c *Client) GetRecentTracks(ctx context.Context, limit int) ([]models.Track, error) {
	if limit <= 0 {
		limit = 50
	}

	var resp subsonicResponse
	params := c.authParams()
	params.Set("type", "newest")
	params.Set("size", fmt.Sprintf("%d", limit))

	if err := c.doRequest(ctx, "getAlbumList2", params, &resp); err != nil {
		return nil, err
	}

	tracks := make([]models.Track, 0, limit)
	if resp.AlbumList2 == nil {
		return tracks, nil
	}

	for _, a := range resp.AlbumList2.Album {
		albumTracks, err := c.GetTracks(ctx, a.ID)
		if err != nil {
			continue
		}
		for _, t := range albumTracks {
			tracks = append(tracks, t)
			if len(tracks) >= limit {
				return tracks, nil
			}
		}
	}

	return tracks, nil
}

func (c *Client) GetArtists(ctx context.Context) ([]models.Artist, error) {
	var resp subsonicResponse
	params := c.authParams()
	if err := c.doRequest(ctx, "getArtists", params, &resp); err != nil {
		return nil, err
	}

	var artists []models.Artist
	if resp.Artists == nil {
		return artists, nil
	}

	for _, idx := range resp.Artists.Index {
		for _, a := range idx.Artist {
			artists = append(artists, models.Artist{
				ID:         a.ID,
				Name:       a.Name,
				AlbumCount: a.AlbumCount,
				CoverURL:   c.getCoverURL(a.CoverArt),
			})
		}
	}

	return artists, nil
}

func (c *Client) GetAlbums(ctx context.Context, artistID string) ([]models.Album, error) {
	var resp subsonicResponse
	params := c.authParams()
	params.Set("id", artistID)

	if err := c.doRequest(ctx, "getArtist", params, &resp); err != nil {
		return nil, err
	}

	var albums []models.Album
	if resp.Artist == nil {
		return albums, nil
	}

	for _, a := range resp.Artist.Album {
		albums = append(albums, models.Album{
			ID:         a.ID,
			Name:       a.Name,
			Artist:     a.Artist,
			ArtistID:   a.ArtistID,
			Year:       a.Year,
			CoverURL:   c.getCoverURL(a.CoverArt),
			TrackCount: a.SongCount,
		})
	}

	return albums, nil
}

func (c *Client) GetTracks(ctx context.Context, albumID string) ([]models.Track, error) {
	var resp subsonicResponse
	params := c.authParams()
	params.Set("id", albumID)

	if err := c.doRequest(ctx, "getAlbum", params, &resp); err != nil {
		return nil, err
	}

	var tracks []models.Track
	if resp.Album == nil {
		return tracks, nil
	}

	for _, s := range resp.Album.Song {
		tracks = append(tracks, models.Track{
			ID:        s.ID,
			Title:     s.Title,
			Artist:    s.Artist,
			ArtistID:  s.ArtistID,
			Album:     s.Album,
			AlbumID:   s.AlbumID,
			Duration:  s.Duration,
			TrackNum:  s.Track,
			StreamURL: c.getStreamURL(s.ID),
			CoverURL:  c.getCoverURL(s.CoverArt),
		})
	}

	return tracks, nil
}

func (c *Client) GetPlaylists(ctx context.Context) ([]models.Playlist, error) {
	var resp subsonicResponse
	params := c.authParams()

	if err := c.doRequest(ctx, "getPlaylists", params, &resp); err != nil {
		return nil, err
	}

	var playlists []models.Playlist
	if resp.Playlists == nil {
		return playlists, nil
	}

	for _, pl := range resp.Playlists.Playlist {
		playlists = append(playlists, models.Playlist{
			ID:         pl.ID,
			Name:       pl.Name,
			TrackCount: pl.SongCount,
			CoverURL:   c.getCoverURL(pl.CoverArt),
		})
	}

	return playlists, nil
}

func (c *Client) GetPlaylistTracks(ctx context.Context, playlistID string) ([]models.Track, error) {
	var resp subsonicResponse
	params := c.authParams()
	params.Set("id", playlistID)

	if err := c.doRequest(ctx, "getPlaylist", params, &resp); err != nil {
		return nil, err
	}

	var tracks []models.Track
	if resp.Playlist == nil {
		return tracks, nil
	}

	for _, e := range resp.Playlist.Entry {
		tracks = append(tracks, models.Track{
			ID:        e.ID,
			Title:     e.Title,
			Artist:    e.Artist,
			ArtistID:  e.ArtistID,
			Album:     e.Album,
			AlbumID:   e.AlbumID,
			Duration:  e.Duration,
			TrackNum:  e.Track,
			StreamURL: c.getStreamURL(e.ID),
			CoverURL:  c.getCoverURL(e.CoverArt),
		})
	}

	return tracks, nil
}

func (c *Client) Search(ctx context.Context, query string) (models.SearchResult, error) {
	var resp subsonicResponse
	params := c.authParams()
	params.Set("query", query)
	params.Set("artistCount", "20")
	params.Set("albumCount", "20")
	params.Set("songCount", "50")

	if err := c.doRequest(ctx, "search3", params, &resp); err != nil {
		return models.SearchResult{}, err
	}

	result := models.SearchResult{}

	if resp.SearchResult3 == nil {
		return result, nil
	}

	for _, a := range resp.SearchResult3.Artist {
		result.Artists = append(result.Artists, models.Artist{
			ID:         a.ID,
			Name:       a.Name,
			AlbumCount: a.AlbumCount,
			CoverURL:   c.getCoverURL(a.CoverArt),
		})
	}

	for _, a := range resp.SearchResult3.Album {
		result.Albums = append(result.Albums, models.Album{
			ID:       a.ID,
			Name:     a.Name,
			Artist:   a.Artist,
			ArtistID: a.ArtistID,
			Year:     a.Year,
			CoverURL: c.getCoverURL(a.CoverArt),
		})
	}

	for _, s := range resp.SearchResult3.Song {
		result.Tracks = append(result.Tracks, models.Track{
			ID:        s.ID,
			Title:     s.Title,
			Artist:    s.Artist,
			ArtistID:  s.ArtistID,
			Album:     s.Album,
			AlbumID:   s.AlbumID,
			Duration:  s.Duration,
			TrackNum:  s.Track,
			StreamURL: c.getStreamURL(s.ID),
			CoverURL:  c.getCoverURL(s.CoverArt),
		})
	}

	return result, nil
}

func (c *Client) StreamTrack(ctx context.Context, trackID string) (io.ReadCloser, error) {
	u, err := url.Parse(c.baseURL + "/rest/stream")
	if err != nil {
		return nil, err
	}

	q := c.authParams()
	q.Set("id", trackID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

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
	u, _ := url.Parse(c.baseURL + "/rest/stream")
	q := c.authParams()
	q.Set("id", trackID)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) GetCoverURL(albumID string) string {
	return c.getCoverURL(albumID)
}

func (c *Client) getCoverURL(coverArtID string) string {
	if coverArtID == "" {
		return ""
	}
	u, _ := url.Parse(c.baseURL + "/rest/getCoverArt")
	q := c.authParams()
	q.Set("id", coverArtID)
	u.RawQuery = q.Encode()
	return u.String()
}
