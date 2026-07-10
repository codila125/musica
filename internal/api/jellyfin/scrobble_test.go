package jellyfin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codila125/musica/internal/config"
)

func TestScrobbleMarksItemPlayed(t *testing.T) {
	var gotMethod, gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/Users/user-1/PlayedItems/track-1", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(config.ServerConfig{URL: server.URL})
	c.userID = "user-1"
	c.apiKey = "key"

	if err := c.Scrobble(t.Context(), "track-1"); err != nil {
		t.Fatalf("Scrobble: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/Users/user-1/PlayedItems/track-1" {
		t.Fatalf("path = %q", gotPath)
	}
}
