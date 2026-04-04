package models

type Track struct {
	ID        string
	Title     string
	Artist    string
	ArtistID  string
	Album     string
	AlbumID   string
	Duration  int
	TrackNum  int
	StreamURL string
	CoverURL  string
	Format    string
}

type Album struct {
	ID         string
	Name       string
	Artist     string
	ArtistID   string
	Year       int
	CoverURL   string
	TrackCount int
}

type Artist struct {
	ID         string
	Name       string
	AlbumCount int
	CoverURL   string
}

type Playlist struct {
	ID         string
	Name       string
	TrackCount int
	CoverURL   string
}

type SearchResult struct {
	Tracks  []Track
	Albums  []Album
	Artists []Artist
}

type PlayerState int

const (
	StateStopped PlayerState = iota
	StatePlaying
	StatePaused
)
