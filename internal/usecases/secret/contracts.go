package secret

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nickolay3132/swarmsec/internal/encryption"
	"github.com/nickolay3132/swarmsec/internal/p2p"
)

type cipher interface {
	Encrypt([]byte) (*encryption.Payload, error)
	Decrypt(*encryption.Payload) ([]byte, error)
}

type sharer interface {
	Split([]byte, int, int) ([][]byte, error)
	Combine([][]byte) ([]byte, error)
}

type network interface {
	Put(ctx context.Context, name string, cipherText []byte, shares [][]byte) ([]peer.AddrInfo, error)
	Get(ctx context.Context, name string, peers []peer.AddrInfo, minShares int) (*p2p.Secret, error)
}

type repository interface {
	SaveShardOwners(ctx context.Context, name string, owners []peer.AddrInfo) error
	GetShardOwners(ctx context.Context, name string) ([]peer.AddrInfo, error)
}
