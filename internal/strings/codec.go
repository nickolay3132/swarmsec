package strings

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type Codec struct{}

func NewCodec() *Codec {
	return &Codec{}
}

func (c *Codec) Parse(str string) (secretName string, cipherText, shard []byte, err error) {
	if len(str) < 3 {
		return "", nil, nil, errors.New("input string too short")
	}

	openBracket := strings.IndexByte(str, '[')
	closeBracket := strings.IndexByte(str, ']')
	if openBracket == -1 || closeBracket == -1 || openBracket > closeBracket {
		return "", nil, nil, errors.New("invalid string format")
	}

	secretName = str[openBracket+1 : closeBracket]
	if secretName == "" {
		return "", nil, nil, errors.New("empty secret name")
	}

	rest := str[closeBracket+1:]
	parts := strings.Split(rest, "&")
	if len(parts) != 2 {
		return "", nil, nil, fmt.Errorf("expected exactly 2 parts separated by '&', got %d", len(parts))
	}

	cipherHex := parts[0]
	shardHex := parts[1]

	if cipherHex == "" {
		return "", nil, nil, errors.New("empty ciphertext hex")
	}
	if shardHex == "" {
		return "", nil, nil, errors.New("empty shard hex")
	}

	cipherText, err = hex.DecodeString(cipherHex)
	if err != nil {
		return "", nil, nil, fmt.Errorf("invalid ciphertext hex: %w", err)
	}

	shard, err = hex.DecodeString(shardHex)
	if err != nil {
		return "", nil, nil, fmt.Errorf("invalid shard hex: %w", err)
	}

	return secretName, cipherText, shard, nil
}

func (c *Codec) Format(secretName string, cipherText, shard []byte) string {
	return fmt.Sprintf("[%s]%s&%s",
		secretName,
		hex.EncodeToString(cipherText),
		hex.EncodeToString(shard))
}
