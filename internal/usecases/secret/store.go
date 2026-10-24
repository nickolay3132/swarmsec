package secret

import (
	"context"
	"fmt"
)

type StoreUC struct {
	cipher  cipher
	sharer  sharer
	network network
	repo    repository
}

func (uc *StoreUC) Execute(ctx context.Context, secret string, secretName string) error {
	payload, err := uc.cipher.Encrypt([]byte(secret))
	if err != nil {
		return fmt.Errorf("could not encrypt secret: %w", err)
	}

	shares, err := uc.sharer.Split(payload.Key, totalShards, quorum)
	if err != nil {
		return fmt.Errorf("could not split secret: %w", err)
	}

	peerInfos, err := uc.network.Put(ctx, secretName, payload.Ciphertext, shares)
	if err != nil {
		return fmt.Errorf("could not put secret: %w", err)
	}

	err = uc.repo.SaveShardOwners(ctx, secretName, peerInfos)
	if err != nil {
		return fmt.Errorf("could not save secret: %w", err)
	}

	return nil
}
