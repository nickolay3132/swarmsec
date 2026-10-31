package database

import (
	"context"
	"fmt"
	"sync"
)

type secret struct {
	Ciphertext []byte
	Shard      []byte
}

type ShardsInMemory struct {
	mu    sync.RWMutex
	store map[string]secret
}

func NewShardsInMemory() *ShardsInMemory {
	return &ShardsInMemory{
		store: make(map[string]secret),
	}
}

func (r *ShardsInMemory) SaveShard(ctx context.Context, secretName string, ciphertext, shard []byte) error {
	ctCopy := make([]byte, len(ciphertext))
	copy(ctCopy, ciphertext)

	shCopy := make([]byte, len(shard))
	copy(shCopy, shard)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[secretName] = secret{
		Ciphertext: ctCopy,
		Shard:      shCopy,
	}
	return nil
}

func (r *ShardsInMemory) GetShard(ctx context.Context, secretName string) (cipherText, shard []byte, err error) {
	r.mu.RLock()
	sec, ok := r.store[secretName]
	r.mu.RUnlock()

	if !ok {
		return nil, nil, fmt.Errorf("secret '%s' not found", secretName)
	}

	ctCopy := make([]byte, len(sec.Ciphertext))
	copy(ctCopy, sec.Ciphertext)

	shCopy := make([]byte, len(sec.Shard))
	copy(shCopy, sec.Shard)

	return ctCopy, shCopy, nil
}
