package p2pnew

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
)

type rendezvousManager struct {
	disc discovery.Discovery
}

func newRendezvousManager(disc discovery.Discovery) *rendezvousManager {
	return &rendezvousManager{
		disc: disc,
	}
}

func (r *rendezvousManager) Advertise(ctx context.Context, namespace string, ttl time.Duration) (time.Duration, error) {
	return r.disc.Advertise(ctx, namespace, discovery.TTL(ttl))
}

func (r *rendezvousManager) AdvertiseLoop(ctx context.Context, namespace string, ttl time.Duration) (time.Duration, error) {
	return ttl, nil
}

func (r *rendezvousManager) FindPeers(ctx context.Context, namespace string, n int) ([]peer.AddrInfo, error) {
	ch, err := r.disc.FindPeers(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("cannot find peers: %w", err)
	}

	peers := make([]peer.AddrInfo, 0, n)
	for p := range ch {
		peers = append(peers, p)
		if len(peers) == n {
			break
		}
	}

	return peers, nil
}
