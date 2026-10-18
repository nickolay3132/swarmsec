package p2pnew

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
)

type ContentManager struct {
	router routing.ContentRouting
}

func NewContentManager(router routing.ContentRouting) *ContentManager {
	return &ContentManager{router: router}
}

func (c *ContentManager) Provide(ctx context.Context, cid cid.Cid, bcast bool) error {
	return c.router.Provide(ctx, cid, bcast)
}

func (c *ContentManager) FindProviders(ctx context.Context, cid cid.Cid, n int) ([]peer.AddrInfo, error) {
	ch := c.router.FindProvidersAsync(ctx, cid, n)

	providers := make([]peer.AddrInfo, 0, n)
	for pi := range ch {
		providers = append(providers, pi)
		if len(providers) >= n {
			break
		}
	}
	return providers, nil
}
