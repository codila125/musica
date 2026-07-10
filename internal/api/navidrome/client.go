package navidrome

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/config"
)

var _ api.Client = (*Client)(nil)

type Client struct {
	baseURL string
	user    string
	token   string
	salt    string
	client  *http.Client
}

func New(cfg config.ServerConfig) *Client {
	salt := fmt.Sprintf("%d", time.Now().UnixNano())
	token := fmt.Sprintf("%x", md5.Sum([]byte(cfg.Password+salt)))

	return &Client{
		baseURL: cfg.URL,
		user:    cfg.Username,
		token:   token,
		salt:    salt,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) authParams() url.Values {
	return url.Values{
		"u": {c.user},
		"t": {c.token},
		"s": {c.salt},
		"v": {"1.16.1"},
		"c": {"musica"},
		"f": {"json"},
	}
}

func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values, v interface{}) error {
	u, err := url.Parse(c.baseURL + "/rest/" + endpoint)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	for k, vs := range params {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var wrapper struct {
		SubsonicResponse json.RawMessage `json:"subsonic-response"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return fmt.Errorf("unmarshal wrapper: %w", err)
	}

	return json.Unmarshal(wrapper.SubsonicResponse, v)
}

func (c *Client) Scrobble(ctx context.Context, trackID string) error {
	params := c.authParams()
	params.Set("id", trackID)
	params.Set("submission", "true")
	var resp struct {
		Status string `json:"status"`
	}
	if err := c.doRequest(ctx, "scrobble", params, &resp); err != nil {
		return api.Wrap(api.ErrorKindNetwork, "scrobble", err)
	}
	return nil
}

func (c *Client) StreamTrack(ctx context.Context, trackID string) (io.ReadCloser, error) {
	u, err := url.Parse(c.baseURL + "/rest/stream")
	if err != nil {
		return nil, err
	}

	q := c.authParams()
	q.Set("id", trackID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c *Client) GetStreamURL(trackID string) string {
	return c.getStreamURL(trackID)
}

func (c *Client) getStreamURL(trackID string) string {
	u, _ := url.Parse(c.baseURL + "/rest/stream")
	q := c.authParams()
	q.Set("id", trackID)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) GetCoverURL(albumID string) string {
	return c.getCoverURL(albumID)
}

func (c *Client) getCoverURL(coverArtID string) string {
	if coverArtID == "" {
		return ""
	}
	u, _ := url.Parse(c.baseURL + "/rest/getCoverArt")
	q := c.authParams()
	q.Set("id", coverArtID)
	u.RawQuery = q.Encode()
	return u.String()
}
