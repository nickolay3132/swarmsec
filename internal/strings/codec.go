package strings

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
)

type Codec struct{}

func (c *Codec) ParsePutString(ctx context.Context, putString string) (secretName string, cipherText, shard []byte, err error) {
	open := strings.IndexByte(putString, '[')
	cloze := strings.IndexByte(putString, ']')
	if open != 0 || cloze == 1 || open > cloze {
		return "", nil, nil, fmt.Errorf("invalid put string format")
	}

	method := putString[open+1 : cloze]
	if strings.ToLower(method) != "put" {
		return "", nil, nil, fmt.Errorf("invalid put string method")
	}

	rest := putString[cloze+1:]
	if rest == "" {
		return "", nil, nil, fmt.Errorf("invalid put string rest")
	}
	args := strings.Split(rest, "&")

	cipherText, err = hex.DecodeString(args[1])
	if err != nil {
		return "", nil, nil, fmt.Errorf("invalid put string cipherText: %v", err)
	}

	shard, err = hex.DecodeString(args[2])
	if err != nil {
		return "", nil, nil, fmt.Errorf("invalid put string shard: %v", err)
	}

	return args[0], cipherText, shard, nil
}
func (c *Codec) FormatProvideString(secretName string, cipherText, shard []byte) string {
	return fmt.Sprintf("[PROVIDE]%s&%s&%s", secretName, hex.EncodeToString(cipherText), hex.EncodeToString(shard))
}
