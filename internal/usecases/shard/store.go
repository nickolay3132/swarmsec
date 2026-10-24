package shard

import (
	"context"
	"errors"
	"fmt"
)

type StoreUC struct {
	repo  repo
	codec codec
}

func NewStoreUC(r repo, p codec) *StoreUC {
	return &StoreUC{
		repo:  r,
		codec: p,
	}
}

func (uc *StoreUC) Execute(ctx context.Context, putString string) error {
	if putString == "" {
		return errors.New("shard: empty put string")
	}

	secretName, cipherText, shard, err := uc.codec.ParsePutString(ctx, putString)
	if err != nil {
		return fmt.Errorf("shard: parse put string: %w", err)
	}

	if secretName == "" {
		return errors.New("shard: empty secret name")
	}

	if err := uc.repo.SaveShard(ctx, secretName, cipherText, shard); err != nil {
		return fmt.Errorf("shard: save shard %q: %w", secretName, err)
	}

	return nil
}
