package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/logger"
)

var _ api.Client = (*Client)(nil)

type Client struct {
	baseURL string
	userID  string
	apiKey  string
	client  *http.Client
}

func New(cfg config.ServerConfig) *Client {
	return &Client{
		baseURL: cfg.URL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Authenticate(ctx context.Context, username, password string) error {
	return c.authenticate(ctx, username, password)
}

func (c *Client) authenticate(ctx context.Context, username, password string) error {
	payload := map[string]string{
		"Username": username,
		"Pw":       password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoints := []string{
		"/Users/Authenticate",
		"/Users/AuthenticateByName",
	}

	var lastErr error
	for _, endpoint := range endpoints {
		u := c.baseURL + endpoint

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(body))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Emby-Authorization", `MediaBrowser Client="musica", Version="0.1.0", DeviceId="musica-tui", Device="Terminal"`)

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("auth failed (HTTP %d): %s", resp.StatusCode, string(respBody))
			continue
		}

		var authResp struct {
			User struct {
				ID string `json:"Id"`
			} `json:"User"`
			AccessToken string `json:"AccessToken"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			lastErr = fmt.Errorf("parse auth response: %w", err)
			continue
		}

		c.userID = authResp.User.ID
		c.apiKey = authResp.AccessToken
		return nil
	}

	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("no compatible auth endpoint found")
}

func (c *Client) authHeader() http.Header {
	h := make(http.Header)
	h.Set("X-Emby-Token", c.apiKey)
	h.Set("X-Emby-Authorization", fmt.Sprintf(`MediaBrowser Client="musica", Version="0.1.0", UserId="%s"`, c.userID))
	return h
}

func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values, v interface{}) error {
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	if params != nil {
		q := u.Query()
		for k, vs := range params {
			for _, v := range vs {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}

	logger.Get().Debug("GET %s", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header = c.authHeader()

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := http.Get(c.baseURL + "/System/Info/Public")
	return err
}

func (c *Client) Scrobble(ctx context.Context, trackID string) error {
	u := fmt.Sprintf("%s/Users/%s/PlayedItems/%s", c.baseURL, c.userID, trackID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return err
	}
	req.Header = c.authHeader()

	resp, err := c.client.Do(req)
	if err != nil {
		return api.Wrap(api.ErrorKindNetwork, "scrobble", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return api.Wrap(api.ErrorKindNetwork, "scrobble", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)))
	}
	return nil
}

func (c *Client) StreamTrack(ctx context.Context, trackID string) (io.ReadCloser, error) {
	u := c.baseURL + fmt.Sprintf("/Items/%s/Download?api_key=%s", trackID, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header = c.authHeader()

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
	return c.baseURL + fmt.Sprintf("/Items/%s/Download?api_key=%s", trackID, c.apiKey)
}

func (c *Client) GetCoverURL(albumID string) string {
	return c.getCoverURL(albumID)
}

func (c *Client) getCoverURL(itemID string) string {
	if itemID == "" {
		return ""
	}
	return fmt.Sprintf("%s/Items/%s/Images/Primary?api_key=%s", c.baseURL, itemID, c.apiKey)
}
