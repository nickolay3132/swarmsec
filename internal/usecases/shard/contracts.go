package shard

import "context"

type repo interface {
	SaveShard(ctx context.Context, secretName string, cipherText, shard []byte) error
	GetShard(ctx context.Context, secretName string) (cipherText, shard []byte, err error)
}

type codec interface {
	Parse(str string) (secretName string, cipherText, shard []byte, err error)
	Format(secretName string, cipherText, shard []byte) string
}
