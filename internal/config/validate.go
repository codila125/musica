package config

import (
	"fmt"
	"net/url"
	"strings"
)

func (c *Config) Normalize() {
	if c.Version <= 0 {
		c.Version = CurrentVersion
	}
	for i := range c.Servers {
		c.Servers[i].Type = strings.ToLower(strings.TrimSpace(c.Servers[i].Type))
		c.Servers[i].Name = strings.TrimSpace(c.Servers[i].Name)
		c.Servers[i].URL = strings.TrimRight(strings.TrimSpace(c.Servers[i].URL), "/")
		c.Servers[i].Username = strings.TrimSpace(c.Servers[i].Username)
	}
	if c.DefaultServer != "" {
		c.DefaultServer = strings.TrimSpace(c.DefaultServer)
	}
}

func (c *Config) Validate() error {
	if c.Version <= 0 {
		return fmt.Errorf("invalid config version: %d", c.Version)
	}

	if len(c.Servers) > 0 && c.DefaultServer == "" {
		return fmt.Errorf("default server is required when servers are configured")
	}

	nameSeen := map[string]struct{}{}
	foundDefault := c.DefaultServer == ""

	for i, s := range c.Servers {
		if s.Type != "navidrome" && s.Type != "jellyfin" {
			return fmt.Errorf("server[%d]: invalid type: %s", i, s.Type)
		}
		if s.Name == "" {
			return fmt.Errorf("server[%d]: name is required", i)
		}
		if _, ok := nameSeen[s.Name]; ok {
			return fmt.Errorf("server[%d]: duplicate server name: %s", i, s.Name)
		}
		nameSeen[s.Name] = struct{}{}

		if s.URL == "" {
			return fmt.Errorf("server[%d]: URL is required", i)
		}
		u, err := url.ParseRequestURI(s.URL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("server[%d]: invalid URL: %s", i, s.URL)
		}

		if s.Username == "" {
			return fmt.Errorf("server[%d]: username is required", i)
		}
		if s.Password == "" {
			return fmt.Errorf("server[%d]: password is required", i)
		}

		if s.Name == c.DefaultServer {
			foundDefault = true
		}
	}

	if !foundDefault {
		return fmt.Errorf("default server %q not found in servers list", c.DefaultServer)
	}

	return nil
}
