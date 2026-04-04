package api

import (
	"context"
	"io"

	"github.com/codila125/musica/internal/models"
)

type Client interface {
	Ping(ctx context.Context) error
	GetRecentTracks(ctx context.Context, limit int) ([]models.Track, error)
	GetArtists(ctx context.Context) ([]models.Artist, error)
	GetAlbums(ctx context.Context, artistID string) ([]models.Album, error)
	GetTracks(ctx context.Context, albumID string) ([]models.Track, error)
	GetPlaylists(ctx context.Context) ([]models.Playlist, error)
	GetPlaylistTracks(ctx context.Context, playlistID string) ([]models.Track, error)
	Search(ctx context.Context, query string) (models.SearchResult, error)
	StreamTrack(ctx context.Context, trackID string) (io.ReadCloser, error)
	GetStreamURL(trackID string) string
	GetCoverURL(albumID string) string
}
