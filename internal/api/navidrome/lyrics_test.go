package navidrome

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/models"
)

func TestGetLyricsSyncedViaOpenSubsonic(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/getLyricsBySongId", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") != "track-1" {
			t.Errorf("id = %q, want track-1", r.URL.Query().Get("id"))
		}
		writeSubsonicResponse(w, map[string]any{
			"lyricsList": map[string]any{
				"structuredLyrics": []map[string]any{
					{
						"synced": true,
						"line": []map[string]any{
							{"start": 0, "value": "First line"},
							{"start": 5200, "value": "Second line"},
						},
					},
				},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(config.ServerConfig{URL: server.URL, Username: "u", Password: "p"})
	got, err := c.GetLyrics(t.Context(), models.Track{ID: "track-1", Artist: "A", Title: "T"})
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

func TestGetLyricsFallsBackToPlainText(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/getLyricsBySongId", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	mux.HandleFunc("/rest/getLyrics", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("artist") != "A" || r.URL.Query().Get("title") != "T" {
			t.Errorf("artist/title = %q/%q", r.URL.Query().Get("artist"), r.URL.Query().Get("title"))
		}
		writeSubsonicResponse(w, map[string]any{
			"lyrics": map[string]any{"artist": "A", "title": "T", "value": "line one\nline two"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(config.ServerConfig{URL: server.URL, Username: "u", Password: "p"})
	got, err := c.GetLyrics(t.Context(), models.Track{ID: "track-1", Artist: "A", Title: "T"})
	if err != nil {
		t.Fatalf("GetLyrics: %v", err)
	}
	if got.Synced {
		t.Fatal("plain lyrics must not be synced")
	}
	if len(got.Lines) != 2 || got.Lines[0].Text != "line one" {
		t.Fatalf("lines = %+v", got.Lines)
	}
}
