package navidrome

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codila125/musica/internal/config"
)

func TestScrobbleHitsSubsonicEndpoint(t *testing.T) {
	var gotID, gotSubmission string
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/scrobble", func(w http.ResponseWriter, r *http.Request) {
		gotID = r.URL.Query().Get("id")
		gotSubmission = r.URL.Query().Get("submission")
		writeSubsonicResponse(w, map[string]any{"status": "ok"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(config.ServerConfig{URL: server.URL, Username: "u", Password: "p"})
	if err := c.Scrobble(t.Context(), "track-1"); err != nil {
		t.Fatalf("Scrobble: %v", err)
	}
	if gotID != "track-1" || gotSubmission != "true" {
		t.Fatalf("scrobble request id=%q submission=%q, want track-1/true", gotID, gotSubmission)
	}
}
