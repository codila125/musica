package jellyfin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codila125/musica/internal/config"
)

// Modern Jellyfin (10.9+) omits AlbumId and MediaSources from /Items
// unless requested via Fields, and audio items carry their own Primary
// image tag. The adapter must request the fields and fall back to
// ParentId / the item's own image.
func TestGetRecentTracksModernJellyfinShape(t *testing.T) {
	var gotFields string
	mux := http.NewServeMux()
	mux.HandleFunc("/Items", func(w http.ResponseWriter, r *http.Request) {
		gotFields = r.URL.Query().Get("Fields")
		json.NewEncoder(w).Encode(map[string]any{
			"Items": []map[string]any{{
				"Id":           "track-1",
				"Name":         "Song",
				"Album":        "Album Name",
				"ParentId":     "album-9",
				"RunTimeTicks": 1200000000,
				"ImageTags":    map[string]string{"Primary": "tag123"},
				"MediaSources": []map[string]any{{"Container": "flac"}},
			}},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(config.ServerConfig{URL: server.URL})
	c.userID = "user-1"
	c.apiKey = "key"

	tracks, err := c.GetRecentTracks(t.Context(), 10)
	if err != nil {
		t.Fatalf("GetRecentTracks: %v", err)
	}
	if !strings.Contains(gotFields, "ParentId") || !strings.Contains(gotFields, "MediaSources") {
		t.Fatalf("Fields = %q, must request ParentId and MediaSources", gotFields)
	}
	if len(tracks) != 1 {
		t.Fatalf("tracks = %d, want 1", len(tracks))
	}
	tr := tracks[0]
	if tr.AlbumID != "album-9" {
		t.Fatalf("AlbumID = %q, want ParentId fallback album-9", tr.AlbumID)
	}
	if !strings.Contains(tr.CoverURL, "/Items/track-1/Images/Primary") {
		t.Fatalf("CoverURL = %q, want the item's own Primary image", tr.CoverURL)
	}
	if tr.Format != "FLAC" {
		t.Fatalf("Format = %q, want FLAC", tr.Format)
	}
}
