package secret

import (
	"context"
	"fmt"

	"github.com/nickolay3132/swarmsec/internal/encryption"
)

type RetrieveUC struct {
	repo    repository
	network network
	cipher  cipher
	sharer  sharer
}

func (uc *RetrieveUC) Execute(ctx context.Context, secretName string) ([]byte, error) {
	owners, err := uc.repo.GetShardOwners(ctx, secretName)
	if err != nil {
		return nil, fmt.Errorf("retrieve shard owners: %w", err)
	}

	secret, err := uc.network.Get(ctx, secretName, owners, quorum)
	if err != nil {
		return nil, fmt.Errorf("retrieve secret: %w", err)
	}

	key, err := uc.sharer.Combine(secret.Shards)
	if err != nil {
		return nil, fmt.Errorf("combine sharers: %w", err)
	}

	s, err := uc.cipher.Decrypt(&encryption.Payload{
		Key:        key,
		Ciphertext: secret.Ciphertext,
	})
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}

	return s, nil
}
