package usecases

import (
	"context"
	"fmt"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nickolay3132/swarmsec/internal/encryption"
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
	FindPeers(context.Context, int) ([]peer.AddrInfo, error)
	GetPK(peer.ID) (ic.PubKey, error)
}

type StoreSecretUC struct {
	cipher  cipher
	sharer  sharer
	network network
}

func (uc *StoreSecretUC) execute(ctx context.Context, secret string) error {
	payload, err := uc.cipher.Encrypt([]byte(secret))
	if err != nil {
		return fmt.Errorf("could not encrypt secret: %w", err)
	}

	shares, err := uc.sharer.Split(payload.Key, 7, 4)
	if err != nil {
		return fmt.Errorf("could not split secret: %w", err)
	}

	fmt.Println("shares", shares)

	peers, err := uc.network.FindPeers(ctx, 7)
	if err != nil {
		return fmt.Errorf("could not find peers: %w", err)
	}

	for _, p := range peers {
		pk, err := uc.network.GetPK(p.ID)
		if err != nil {
			return fmt.Errorf("could not get public key: %w", err)
		}

		pk.Type()
	}

	return nil
}
