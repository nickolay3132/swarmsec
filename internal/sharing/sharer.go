package sharing

import "go.bryk.io/pkg/crypto/shamir"

type Sharer struct{}

func NewSharer() *Sharer {
	return &Sharer{}
}

func (s *Sharer) Split(secret []byte, parts, threshold int) ([][]byte, error) {
	return shamir.Split(secret, parts, threshold)
}

func (s *Sharer) Combine(parts [][]byte) ([]byte, error) {
	return shamir.Combine(parts)
}
