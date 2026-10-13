package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type Payload struct {
	Key        []byte
	Ciphertext []byte
}

type Cipher struct{}

func NewCipher() *Cipher {
	return &Cipher{}
}

func (c *Cipher) Encrypt(plaintext []byte) (*Payload, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	gcm, err := c.prepareGCM(key)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return &Payload{
		Key:        key,
		Ciphertext: ciphertext,
	}, nil
}

func (c *Cipher) Decrypt(p *Payload) ([]byte, error) {
	nonce, ciphertext := p.Ciphertext[:12], p.Ciphertext[12:]

	gcm, err := c.prepareGCM(p.Key)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (c *Cipher) prepareGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm, nil
}
