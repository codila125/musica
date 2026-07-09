package navidrome

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codila125/musica/internal/config"
)

func TestGetRecentTracksFetchesAlbumsConcurrently(t *testing.T) {
	const albumCount = 10
	const limit = 6

	var inFlight int32
	var maxInFlight int32
	var albumCalls int32

	mux := http.NewServeMux()
	mux.HandleFunc("/rest/getAlbumList2", func(w http.ResponseWriter, r *http.Request) {
		albums := make([]struct {
			ID string `json:"id"`
		}, albumCount)
		for i := range albums {
			albums[i].ID = fmt.Sprintf("album-%d", i)
		}
		writeSubsonicResponse(w, map[string]any{
			"albumList2": map[string]any{"album": albums},
		})
	})
	mux.HandleFunc("/rest/getAlbum", func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&inFlight, 1)
		defer atomic.AddInt32(&inFlight, -1)
		for {
			max := atomic.LoadInt32(&maxInFlight)
			if cur <= max || atomic.CompareAndSwapInt32(&maxInFlight, max, cur) {
				break
			}
		}
		atomic.AddInt32(&albumCalls, 1)
		time.Sleep(20 * time.Millisecond)

		albumID := r.URL.Query().Get("id")
		writeSubsonicResponse(w, map[string]any{
			"album": map[string]any{
				"id": albumID,
				"song": []map[string]any{
					{"id": albumID + "-track", "title": "Song"},
				},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(config.ServerConfig{URL: server.URL, Username: "u", Password: "p"})

	tracks, err := c.GetRecentTracks(t.Context(), limit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != limit {
		t.Fatalf("expected %d tracks, got %d", limit, len(tracks))
	}

	if calls := atomic.LoadInt32(&albumCalls); calls >= albumCount {
		t.Fatalf("expected early exit before fetching all %d albums, made %d calls", albumCount, calls)
	}

	if max := atomic.LoadInt32(&maxInFlight); max < 2 {
		t.Fatalf("expected concurrent album fetches (max in-flight >= 2), got %d", max)
	}
}

func writeSubsonicResponse(w http.ResponseWriter, body map[string]any) {
	inner, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"subsonic-response": %s}`, inner)
}
