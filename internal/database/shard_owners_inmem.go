package database

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

type ShardOwnersInMemory struct {
	mu    sync.RWMutex
	store map[string][]peer.AddrInfo
}

func NewShardOwnersInMemory() *ShardOwnersInMemory {
	return &ShardOwnersInMemory{
		store: make(map[string][]peer.AddrInfo),
	}
}

func (r *ShardOwnersInMemory) SaveShardOwners(ctx context.Context, name string, owners []peer.AddrInfo) error {
	cp := make([]peer.AddrInfo, len(owners))
	copy(cp, owners)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[name] = cp
	return nil
}
func (r *ShardOwnersInMemory) GetShardOwners(ctx context.Context, name string) ([]peer.AddrInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	owners, ok := r.store[name]
	if !ok {
		return make([]peer.AddrInfo, 0), nil
	}

	cp := make([]peer.AddrInfo, len(owners))
	copy(cp, owners)
	return cp, nil
}
