package shard

import (
	"context"
	"errors"
	"fmt"
)

type ProvideUC struct {
	repo  repo
	codec codec
}

func NewProvideUC(repo repo, codec codec) *ProvideUC {
	return &ProvideUC{repo: repo, codec: codec}
}

func (uc *ProvideUC) Execute(ctx context.Context, secretName string) (string, error) {
	if secretName == "" {
		return "", errors.New("provide shard: empty secret name")
	}

	cipherText, shard, err := uc.repo.GetShard(ctx, secretName)
	if err != nil {
		return "", fmt.Errorf("provide shard: get shard %q: %w", secretName, err)
	}

	provideString, err := uc.codec.FormatProvideString(secretName, cipherText, shard)
	if err != nil {
		return "", fmt.Errorf("provide shard: format shard %q: %w", secretName, err)
	}

	return provideString, nil
}
