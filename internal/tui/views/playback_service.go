package views

import (
	"github.com/codila125/musica/internal/models"
)

type PlaybackService interface {
	ToggleTrack(track models.Track) error
	ToggleQueueTrack(queue []models.Track, cursor int) error
	PlayTrack(track models.Track) error
	QueueTrack(track models.Track) error
	Stop() error
	CurrentTrack() *models.Track
	State() models.PlayerState
	Queue() []models.Track
	CurrentIndex() int
}
