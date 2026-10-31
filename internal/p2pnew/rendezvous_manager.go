package p2pnew

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
)

type RendezvousManager struct {
	selfID peer.ID
	disc   discovery.Discovery
}

func NewRendezvousManager(selfID peer.ID, disc discovery.Discovery) *RendezvousManager {
	return &RendezvousManager{
		selfID: selfID,
		disc:   disc,
	}
}

func (r *RendezvousManager) Advertise(ctx context.Context, namespace string, ttl time.Duration) (time.Duration, error) {
	return r.disc.Advertise(ctx, namespace, discovery.TTL(ttl))
}

func (r *RendezvousManager) AdvertiseLoop(ctx context.Context, namespace string, ttl time.Duration) (time.Duration, error) {
	if namespace == "" {
		return 0, errors.New("empty namespace")
	}
	if ttl <= 0 {
		return 0, errors.New("ttl must be > 0")
	}

	go func() {
		interval := ttl / 2
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_, _ = r.Advertise(ctx, namespace, ttl)
			}
		}
	}()
	return ttl, nil
}

func (r *RendezvousManager) FindPeers(ctx context.Context, namespace string, n int) ([]peer.AddrInfo, error) {
	ch, err := r.disc.FindPeers(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("cannot find peers: %w", err)
	}

	peers := make([]peer.AddrInfo, 0, n)
	for p := range ch {
		if p.ID == r.selfID {
			continue
		}
		if len(peers) == n {
			break
		}

		peers = append(peers, p)
	}

	return peers, nil
}
