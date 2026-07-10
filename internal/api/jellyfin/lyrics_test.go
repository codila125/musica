package jellyfin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/models"
)

func TestGetLyricsSynced(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/Audio/track-1/Lyrics", func(w http.ResponseWriter, r *http.Request) {
		// Jellyfin reports offsets in ticks: 10,000 ticks per millisecond.
		json.NewEncoder(w).Encode(map[string]any{
			"Lyrics": []map[string]any{
				{"Text": "First line", "Start": 0},
				{"Text": "Second line", "Start": 52000000},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(config.ServerConfig{URL: server.URL})
	c.userID = "user-1"
	c.apiKey = "key"

	got, err := c.GetLyrics(t.Context(), models.Track{ID: "track-1"})
	if err != nil {
		t.Fatalf("GetLyrics: %v", err)
	}
	if !got.Synced {
		t.Fatal("want synced lyrics")
	}
	if len(got.Lines) != 2 || got.Lines[1].StartMs != 5200 || got.Lines[1].Text != "Second line" {
		t.Fatalf("lines = %+v", got.Lines)
	}
}
