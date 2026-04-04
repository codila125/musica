//go:build testmpv

package views

import (
	"errors"
	"testing"

	"github.com/codila125/musica/internal/models"
)

func TestSearchModelIgnoresStaleMessages(t *testing.T) {
	pl := &fakePlayerService{}
	m := NewSearchModelWithService(fakeAPIClient{}, pl)

	stale := searchResultsMsg{id: m.searchReqID - 1, result: models.SearchResult{}}
	m2, _ := m.Update(stale)
	if m2.loading {
		t.Fatalf("expected stale message not to affect loading")
	}
}

func TestSearchModelHandlesErrorMessage(t *testing.T) {
	pl := &fakePlayerService{}
	m := NewSearchModelWithService(fakeAPIClient{}, pl)
	m.loading = true
	err := errors.New("search failed")

	m2, _ := m.Update(searchResultsMsg{id: m.searchReqID, err: err})
	if m2.err == nil {
		t.Fatalf("expected search error to be set")
	}
}
