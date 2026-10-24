package shard

import "context"

type repo interface {
	SaveShard(ctx context.Context, secretName string, cipherText, shard []byte) error
	GetShard(ctx context.Context, secretName string) (cipherText, shard []byte, err error)
}

type codec interface {
	ParsePutString(ctx context.Context, putString string) (secretName string, cipherText, shard []byte, err error)
	FormatProvideString(secretName string, cipherText, shard []byte) (string, error)
}
