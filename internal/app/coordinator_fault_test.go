package app

import (
	"context"
	"errors"
	"testing"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/config"
)

func TestCoordinatorPropagatesConnectorFailureKind(t *testing.T) {
	servers := []config.ServerConfig{{Name: "J", Type: "jellyfin"}}
	connector := ConnectorFunc(func(ctx context.Context, serverCfg config.ServerConfig) (api.Client, error) {
		return nil, api.Wrap(api.ErrorKindNetwork, "jellyfin.ping", errors.New("i/o timeout"))
	})

	c := NewCoordinator(servers, connector)
	res := c.ConnectIndex(context.Background(), 0)
	if res.Err == nil {
		t.Fatalf("expected connector error")
	}
	if api.KindOf(res.Err) != api.ErrorKindNetwork {
		t.Fatalf("expected network error kind")
	}
}
