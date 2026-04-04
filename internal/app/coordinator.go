package app

import (
	"context"
	"fmt"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/api/jellyfin"
	"github.com/codila125/musica/internal/api/navidrome"
	"github.com/codila125/musica/internal/config"
)

type Connector interface {
	Connect(ctx context.Context, serverCfg config.ServerConfig) (api.Client, error)
}

type ConnectorFunc func(ctx context.Context, serverCfg config.ServerConfig) (api.Client, error)

func (f ConnectorFunc) Connect(ctx context.Context, serverCfg config.ServerConfig) (api.Client, error) {
	return f(ctx, serverCfg)
}

type Coordinator struct {
	servers   []config.ServerConfig
	connector Connector
}

type SwitchResult struct {
	Client api.Client
	Index  int
	Err    error
}

func NewCoordinator(servers []config.ServerConfig, connector Connector) *Coordinator {
	if connector == nil {
		connector = ConnectorFunc(defaultConnect)
	}
	return &Coordinator{servers: servers, connector: connector}
}

func (c *Coordinator) NextIndex(current int) (int, bool) {
	if len(c.servers) <= 1 {
		return 0, false
	}
	return (current + 1) % len(c.servers), true
}

func (c *Coordinator) ConnectIndex(ctx context.Context, index int) SwitchResult {
	if index < 0 || index >= len(c.servers) {
		return SwitchResult{Err: api.Wrap(api.ErrorKindConfig, "connect.index", fmt.Errorf("invalid server index: %d", index))}
	}

	serverCfg := c.servers[index]
	client, err := c.connector.Connect(ctx, serverCfg)
	if err != nil {
		return SwitchResult{Err: err}
	}

	return SwitchResult{Client: client, Index: index}
}

func defaultConnect(ctx context.Context, serverCfg config.ServerConfig) (api.Client, error) {
	switch serverCfg.Type {
	case "navidrome":
		c := navidrome.New(serverCfg)
		if err := c.Ping(ctx); err != nil {
			return nil, api.Wrap(api.ErrorKindNetwork, "navidrome.ping", err)
		}
		return c, nil
	case "jellyfin":
		c := jellyfin.New(serverCfg)
		if err := c.Ping(ctx); err != nil {
			return nil, api.Wrap(api.ErrorKindNetwork, "jellyfin.ping", err)
		}
		if err := c.Authenticate(ctx, serverCfg.Username, serverCfg.Password); err != nil {
			return nil, api.Wrap(api.ErrorKindAuth, "jellyfin.authenticate", err)
		}
		return c, nil
	default:
		return nil, api.Wrap(api.ErrorKindConfig, "connect.type", fmt.Errorf("unknown server type: %s", serverCfg.Type))
	}
}
