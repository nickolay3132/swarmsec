package usecases

import (
	"context"
	"encoding/hex"
	"fmt"

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
	//GetPK(peer.ID) (ic.PubKey, error)
	StoreString(context.Context, peer.ID, string) error
}

type repository interface {
	SaveSecret(ctx context.Context, name string, owners []peer.ID) error
}

type StoreSecretUC struct {
	cipher  cipher
	sharer  sharer
	network network
	repo    repository
}

func (uc *StoreSecretUC) execute(ctx context.Context, secret string, secretName string) error {
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

	peerIds := make([]peer.ID, 0, len(peers))

	shareId := 0
	for _, p := range peers {
		//pk, err := uc.network.GetPK(p.ID)
		//if err != nil {
		//	return fmt.Errorf("could not get public key: %w", err)
		//}

		err := uc.network.StoreString(ctx, p.ID, fmt.Sprintf(
			"[PUT]%s&%s&%s",
			secretName,
			hex.EncodeToString(shares[shareId]),
			hex.EncodeToString(payload.Ciphertext),
		))
		if err != nil {
			return fmt.Errorf("could not store share: %w", err)
		}

		peerIds = append(peerIds, p.ID)
		shareId++
	}

	err = uc.repo.SaveSecret(ctx, secretName, peerIds)
	if err != nil {
		return fmt.Errorf("could not save secret: %w", err)
	}

	return nil
}
