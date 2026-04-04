package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/codila125/musica/internal/config"
)

func runSetup() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Musica Server Setup ===")
	fmt.Println()

	fmt.Print("Server type (navidrome/jellyfin) [navidrome]: ")
	serverType := readLine(reader)
	if serverType == "" {
		serverType = "navidrome"
	}

	if serverType != "navidrome" && serverType != "jellyfin" {
		return fmt.Errorf("invalid server type: %s", serverType)
	}

	fmt.Print("Server name (e.g. my-music): ")
	name := readLine(reader)
	if name == "" {
		return fmt.Errorf("server name is required")
	}

	fmt.Print("Server URL (e.g. http://localhost:4533): ")
	url := readLine(reader)
	if url == "" {
		return fmt.Errorf("server URL is required")
	}

	url = strings.TrimRight(url, "/")

	fmt.Print("Username: ")
	username := readLine(reader)
	if username == "" {
		return fmt.Errorf("username is required")
	}

	fmt.Print("Password: ")
	password := readPassword(reader)
	if password == "" {
		return fmt.Errorf("password is required")
	}

	server := config.ServerConfig{
		Type:     serverType,
		Name:     name,
		URL:      url,
		Username: username,
		Password: password,
	}

	if len(cfg.Servers) == 0 {
		cfg.DefaultServer = name
	}

	cfg.Servers = append(cfg.Servers, server)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	path, _ := config.ConfigPath()
	fmt.Printf("\n✓ Server '%s' added successfully\n", name)
	fmt.Printf("  Config saved to: %s\n", path)
	fmt.Println()
	fmt.Println("Connect with: musica --server", name)
	fmt.Println("Or set as default in config and run: musica")

	return nil
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

func readPassword(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
