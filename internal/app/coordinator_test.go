package app

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/models"
)

type fakeClient struct{}

func (f fakeClient) Ping(ctx context.Context) error { return nil }
func (f fakeClient) GetRecentTracks(ctx context.Context, limit int) ([]models.Track, error) {
	return nil, nil
}
func (f fakeClient) GetArtists(ctx context.Context) ([]models.Artist, error) { return nil, nil }
func (f fakeClient) GetAlbums(ctx context.Context, artistID string) ([]models.Album, error) {
	return nil, nil
}
func (f fakeClient) GetTracks(ctx context.Context, albumID string) ([]models.Track, error) {
	return nil, nil
}
func (f fakeClient) GetPlaylists(ctx context.Context) ([]models.Playlist, error) { return nil, nil }
func (f fakeClient) GetPlaylistTracks(ctx context.Context, playlistID string) ([]models.Track, error) {
	return nil, nil
}
func (f fakeClient) Search(ctx context.Context, query string) (models.SearchResult, error) {
	return models.SearchResult{}, nil
}
func (f fakeClient) StreamTrack(ctx context.Context, trackID string) (io.ReadCloser, error) {
	return nil, nil
}
func (f fakeClient) GetStreamURL(trackID string) string { return "" }
func (f fakeClient) GetCoverURL(albumID string) string  { return "" }

func TestNextIndex(t *testing.T) {
	servers := []config.ServerConfig{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	c := NewCoordinator(servers, nil)

	next, ok := c.NextIndex(1)
	if !ok || next != 2 {
		t.Fatalf("expected next index 2, got %d (ok=%v)", next, ok)
	}
}

func TestConnectIndexInvalid(t *testing.T) {
	c := NewCoordinator(nil, nil)
	res := c.ConnectIndex(context.Background(), 1)
	if res.Err == nil {
		t.Fatalf("expected error for invalid index")
	}
	if api.KindOf(res.Err) != api.ErrorKindConfig {
		t.Fatalf("expected config error kind")
	}
}

func TestConnectIndexUsesConnector(t *testing.T) {
	servers := []config.ServerConfig{{Name: "a", Type: "navidrome"}}
	connector := ConnectorFunc(func(ctx context.Context, serverCfg config.ServerConfig) (api.Client, error) {
		if serverCfg.Name != "a" {
			return nil, errors.New("wrong server")
		}
		return fakeClient{}, nil
	})

	c := NewCoordinator(servers, connector)
	res := c.ConnectIndex(context.Background(), 0)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.Index != 0 {
		t.Fatalf("expected index 0, got %d", res.Index)
	}
}
