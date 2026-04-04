package views

import "github.com/codila125/musica/internal/models"

type PlayerService interface {
	Play(track models.Track) error
	PlayQueue(tracks []models.Track, startIdx int) error
	Pause() error
	Resume() error
	Stop() error
	CurrentTrack() *models.Track
	State() models.PlayerState
	Queue() []models.Track
	CurrentIndex() int
	AppendToQueue(track models.Track) error
}
