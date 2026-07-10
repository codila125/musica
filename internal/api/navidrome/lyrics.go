package navidrome

import (
	"context"
	"strings"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/models"
)

// GetLyrics prefers OpenSubsonic synced lyrics and falls back to the
// legacy plain-text getLyrics endpoint for older servers.
func (c *Client) GetLyrics(ctx context.Context, track models.Track) (models.Lyrics, error) {
	if lyrics, err := c.getLyricsBySongID(ctx, track.ID); err == nil && len(lyrics.Lines) > 0 {
		return lyrics, nil
	}
	return c.getPlainLyrics(ctx, track)
}

func (c *Client) getLyricsBySongID(ctx context.Context, trackID string) (models.Lyrics, error) {
	params := c.authParams()
	params.Set("id", trackID)

	var resp struct {
		LyricsList struct {
			StructuredLyrics []struct {
				Synced bool `json:"synced"`
				Line   []struct {
					Start int    `json:"start"`
					Value string `json:"value"`
				} `json:"line"`
			} `json:"structuredLyrics"`
		} `json:"lyricsList"`
	}
	if err := c.doRequest(ctx, "getLyricsBySongId", params, &resp); err != nil {
		return models.Lyrics{}, api.Wrap(api.ErrorKindNetwork, "getLyricsBySongId", err)
	}

	for _, sl := range resp.LyricsList.StructuredLyrics {
		if len(sl.Line) == 0 {
			continue
		}
		out := models.Lyrics{Synced: sl.Synced}
		for _, l := range sl.Line {
			out.Lines = append(out.Lines, models.LyricLine{StartMs: l.Start, Text: l.Value})
		}
		return out, nil
	}
	return models.Lyrics{}, nil
}

func (c *Client) getPlainLyrics(ctx context.Context, track models.Track) (models.Lyrics, error) {
	params := c.authParams()
	params.Set("artist", track.Artist)
	params.Set("title", track.Title)

	var resp struct {
		Lyrics struct {
			Value string `json:"value"`
		} `json:"lyrics"`
	}
	if err := c.doRequest(ctx, "getLyrics", params, &resp); err != nil {
		return models.Lyrics{}, api.Wrap(api.ErrorKindNetwork, "getLyrics", err)
	}

	var out models.Lyrics
	for _, line := range strings.Split(resp.Lyrics.Value, "\n") {
		if line = strings.TrimRight(line, "\r"); line != "" {
			out.Lines = append(out.Lines, models.LyricLine{Text: line})
		}
	}
	return out, nil
}
