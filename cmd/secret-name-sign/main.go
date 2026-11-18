package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	msg := []byte("my-secret-name")
	sig := ed25519.Sign(priv, msg)

	fmt.Println(hex.EncodeToString(pub))
	fmt.Println(hex.EncodeToString(sig))
	fmt.Println(ed25519.Verify(pub, msg, sig))
}
