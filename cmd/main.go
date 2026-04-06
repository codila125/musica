package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/api/jellyfin"
	"github.com/codila125/musica/internal/api/navidrome"
	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/player"
	"github.com/codila125/musica/internal/tui"
)

func runList() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Servers) == 0 {
		fmt.Println("No servers configured. Run: musica setup")
		return nil
	}

	fmt.Printf("Default: %s\n\n", cfg.DefaultServer)

	for _, s := range cfg.Servers {
		defaultMark := " "
		if s.Name == cfg.DefaultServer {
			defaultMark = "*"
		}
		fmt.Printf("%s %-20s %-12s %s\n", defaultMark, s.Name, s.Type, s.URL)
	}

	fmt.Println("\n* = default server")
	return nil
}

func runRemove(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	found := false
	var remaining []config.ServerConfig
	for _, s := range cfg.Servers {
		if s.Name == name {
			found = true
			continue
		}
		remaining = append(remaining, s)
	}

	if !found {
		return fmt.Errorf("server %q not found", name)
	}

	cfg.Servers = remaining

	if cfg.DefaultServer == name {
		if len(remaining) > 0 {
			cfg.DefaultServer = remaining[0].Name
		} else {
			cfg.DefaultServer = ""
		}
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("✓ Server '%s' removed\n", name)
	return nil
}

func runPlayer(serverName string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Servers) == 0 {
		fmt.Println("No servers configured. Run: musica setup")
		return nil
	}

	if serverName == "" {
		serverName = cfg.DefaultServer
	}
	if serverName == "" && len(cfg.Servers) > 0 {
		serverName = cfg.Servers[0].Name
	}

	var serverCfg *config.ServerConfig
	for i, s := range cfg.Servers {
		if s.Name == serverName {
			serverCfg = &cfg.Servers[i]
			break
		}
	}

	if serverCfg == nil {
		fmt.Printf("Server %q not found. Available servers:\n", serverName)
		for _, s := range cfg.Servers {
			fmt.Printf("  - %s (%s)\n", s.Name, s.Type)
		}
		return fmt.Errorf("server %q not found", serverName)
	}

	var client interface{}
	switch serverCfg.Type {
	case "navidrome":
		client = navidrome.New(*serverCfg)
	case "jellyfin":
		client = jellyfin.New(*serverCfg)
	default:
		return fmt.Errorf("unknown server type: %s", serverCfg.Type)
	}

	ctx := context.Background()

	if nc, ok := client.(*navidrome.Client); ok {
		if err := nc.Ping(ctx); err != nil {
			return fmt.Errorf("connect to navidrome: %w", err)
		}
		fmt.Printf("Connected to Navidrome: %s\n", serverCfg.Name)
	} else if jc, ok := client.(*jellyfin.Client); ok {
		if err := jc.Ping(ctx); err != nil {
			return fmt.Errorf("connect to jellyfin: %w", err)
		}
		if err := jc.Authenticate(ctx, serverCfg.Username, serverCfg.Password); err != nil {
			return fmt.Errorf("authenticate with jellyfin: %w", err)
		}
		fmt.Printf("Connected to Jellyfin: %s\n", serverCfg.Name)
	}

	pl, err := player.New()
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "mpv/client.h") || strings.Contains(errMsg, "libmpv") {
			return fmt.Errorf("initialize player: %w\nHint: install mpv/libmpv and ensure it is on your library path", err)
		}
		return fmt.Errorf("initialize player: %w", err)
	}
	defer pl.Close()
	go pl.Monitor(nil)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = pl.Stop()
		_ = pl.Close()
	}()

	currentServer := 0
	for i, s := range cfg.Servers {
		if s.Name == serverCfg.Name {
			currentServer = i
			break
		}
	}

	m := tui.NewModel(client.(tui.API), pl, cfg.Servers, currentServer)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}

	return nil
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "setup":
			if err := runSetup(); err != nil {
				log.Fatal(err)
			}
			return
		case "list":
			if err := runList(); err != nil {
				log.Fatal(err)
			}
			return
		case "remove":
			if len(os.Args) < 3 {
				fmt.Println("Usage: musica remove <server-name>")
				os.Exit(1)
			}
			if err := runRemove(os.Args[2]); err != nil {
				log.Fatal(err)
			}
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	serverName := flag.String("server", "", "Server name to connect to")
	flag.Parse()

	if err := runPlayer(*serverName); err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println(`musica - TUI music player for Navidrome & Jellyfin

Usage:
  musica [flags]
  musica <command> [args]

Commands:
  setup              Add a new server interactively
  list               List configured servers
  remove <name>      Remove a server by name
  help               Show this help

Flags:
  --server <name>    Connect to a specific server

Examples:
  musica setup                   Add a server
  musica                         Connect to default server
  musica --server my-music       Connect to a specific server
  musica list                    List all servers`)
}
