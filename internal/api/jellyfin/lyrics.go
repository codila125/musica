package jellyfin

import (
	"context"
	"fmt"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/models"
)

func (c *Client) GetLyrics(ctx context.Context, track models.Track) (models.Lyrics, error) {
	var resp struct {
		Lyrics []struct {
			Text  string `json:"Text"`
			Start int64  `json:"Start"`
		} `json:"Lyrics"`
	}
	endpoint := fmt.Sprintf("/Audio/%s/Lyrics", track.ID)
	if err := c.doRequest(ctx, endpoint, nil, &resp); err != nil {
		return models.Lyrics{}, api.Wrap(api.ErrorKindNetwork, "lyrics", err)
	}

	var out models.Lyrics
	for _, l := range resp.Lyrics {
		// Start is in ticks: 10,000 ticks per millisecond.
		startMs := int(l.Start / 10000)
		if startMs > 0 {
			out.Synced = true
		}
		out.Lines = append(out.Lines, models.LyricLine{StartMs: startMs, Text: l.Text})
	}
	return out, nil
}
