package config

import "testing"

func TestNormalizeAndValidate(t *testing.T) {
	cfg := &Config{
		Servers: []ServerConfig{{
			Type:     "  NAVIDROME  ",
			Name:     "  home  ",
			URL:      "https://example.com/",
			Username: " user ",
			Password: "pass",
		}},
		DefaultServer: "  home  ",
	}

	cfg.Normalize()
	if cfg.Version != CurrentVersion {
		t.Fatalf("expected version %d, got %d", CurrentVersion, cfg.Version)
	}
	if cfg.Servers[0].Type != "navidrome" {
		t.Fatalf("expected normalized type, got %q", cfg.Servers[0].Type)
	}
	if cfg.Servers[0].Name != "home" {
		t.Fatalf("expected trimmed name, got %q", cfg.Servers[0].Name)
	}
	if cfg.Servers[0].URL != "https://example.com" {
		t.Fatalf("expected normalized URL, got %q", cfg.Servers[0].URL)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}
}

func TestValidateRejectsDuplicateNames(t *testing.T) {
	cfg := &Config{
		Version: CurrentVersion,
		Servers: []ServerConfig{
			{Type: "navidrome", Name: "same", URL: "https://a.com", Username: "u", Password: "p"},
			{Type: "jellyfin", Name: "same", URL: "https://b.com", Username: "u", Password: "p"},
		},
		DefaultServer: "same",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected duplicate name validation error")
	}
}

func TestValidateRequiresDefaultServerWhenServersExist(t *testing.T) {
	cfg := &Config{
		Version: CurrentVersion,
		Servers: []ServerConfig{
			{Type: "navidrome", Name: "home", URL: "https://a.com", Username: "u", Password: "p"},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected missing default server validation error")
	}
}

func TestServerRedacted(t *testing.T) {
	s := ServerConfig{Type: "navidrome", Name: "x", URL: "https://x", Username: "u", Password: "secret"}
	r := s.Redacted()
	if r.Password != "***" {
		t.Fatalf("expected redacted password, got %q", r.Password)
	}
}
