package database

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type KeyStoreFS struct{}

type KeyPair struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

func NewKeyStoreFS() *KeyStoreFS {
	return &KeyStoreFS{}
}

func (ks *KeyStoreFS) Save(path string, priv ed25519.PrivateKey) error {
	if len(priv) != ed25519.PrivateKeySize {
		return errors.New("keystore: invalid private key")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("keystore: mkdir: %w", err)
	}

	if err := writeFileAtomic(path, priv, 0600); err != nil {
		return fmt.Errorf("keystore: save private key: %w", err)
	}

	return nil
}

func (ks *KeyStoreFS) Load(path string) (KeyPair, error) {
	privData, err := os.ReadFile(path)
	if err != nil {
		return KeyPair{}, fmt.Errorf("keystore: read private key: %w", err)
	}

	if len(privData) != ed25519.PrivateKeySize {
		return KeyPair{}, errors.New("keystore: invalid private key size")
	}

	priv := ed25519.PrivateKey(privData)
	pub := priv.Public().(ed25519.PublicKey)

	return KeyPair{Private: priv, Public: pub}, nil
}

func (ks *KeyStoreFS) LoadOrCreate(path string) (KeyPair, error) {
	kp, err := ks.Load(path)
	if err == nil {
		return kp, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return KeyPair{}, err
	}

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return KeyPair{}, fmt.Errorf("keystore: generate: %w", err)
	}

	if err := ks.Save(path, priv); err != nil {
		return KeyPair{}, err
	}

	return KeyPair{Private: priv, Public: pub}, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	tmp := path + ".tmp"

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()

	if _, err = f.Write(data); err != nil {
		return err
	}

	if err = f.Sync(); err != nil {
		return err
	}

	if err = f.Close(); err != nil {
		return err
	}

	if err = os.Rename(tmp, path); err != nil {
		return err
	}

	return nil
}
