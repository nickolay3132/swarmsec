package p2pnew

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nickolay3132/swarmsec/internal/p2p"
)

const defaultNamespace = "sws"

type codec interface {
	Parse(str string) (secretName string, cipherText, shard []byte, err error)
	Format(secretName string, cipherText, shard []byte) string
}

type NetworkAdapter struct {
	network *Network
	codec   codec
}

func NewNetworkAdapter(network *Network, c codec) *NetworkAdapter {
	a := &NetworkAdapter{
		network: network,
		codec:   c,
	}
	return a
}

func (a *NetworkAdapter) Put(ctx context.Context, name string, cipherText []byte, shares [][]byte) ([]peer.AddrInfo, error) {
	if name == "" {
		return nil, errors.New("put: empty name")
	}
	if len(shares) == 0 {
		return nil, errors.New("put: no shares")
	}

	peers, err := a.network.FindPeers(ctx, defaultNamespace, len(shares)+1)
	if err != nil {
		return nil, fmt.Errorf("put: find peers: %w", err)
	}
	if len(peers) < len(shares) {
		return nil, fmt.Errorf("put: need %d peers, found %d", len(shares), len(peers))
	}
	peers = peers[:len(shares)]

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		delivered []peer.AddrInfo
		errs      []error
	)
	for i, p := range peers {
		wg.Add(1)
		go func(i int, p peer.AddrInfo) {
			defer wg.Done()

			payload := a.codec.Format(name, cipherText, shares[i])

			if err := a.sendPut(ctx, p, payload); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("peer %s: %w", p.ID, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			delivered = append(delivered, p)
			mu.Unlock()
		}(i, p)
	}
	wg.Wait()

	if len(delivered) < len(shares) {
		return delivered, fmt.Errorf(
			"put: delivered %d/%d shares: %w",
			len(delivered), len(shares), errors.Join(errs...),
		)
	}
	return delivered, nil
}

func (a *NetworkAdapter) sendPut(ctx context.Context, p peer.AddrInfo, payload string) error {
	s, err := a.network.OpenStream(ctx, p, "/swarmsec/1.0.0/put")
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	defer s.Close()

	if _, err := io.WriteString(s, payload+"\n"); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := s.CloseWrite(); err != nil {
		return fmt.Errorf("close write: %w", err)
	}

	br := bufio.NewReader(s)
	resp, err := br.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read response: %w", err)
	}
	resp = strings.TrimRight(resp, "\n")

	if resp != "ok" {
		return fmt.Errorf("peer rejected put: %q", resp)
	}
	return nil
}

// ---------- GET ----------

func (a *NetworkAdapter) Get(ctx context.Context, name string, peers []peer.AddrInfo, minShares int) (*p2p.Secret, error) {
	if minShares <= 0 {
		return nil, errors.New("get: minShares must be > 0")
	}
	if len(peers) == 0 {
		return nil, errors.New("get: no peers")
	}

	type res struct {
		p    peer.AddrInfo
		data string
		err  error
	}
	results := make([]res, len(peers))
	var wg sync.WaitGroup
	for i, p := range peers {
		wg.Add(1)
		go func(i int, p peer.AddrInfo) {
			defer wg.Done()
			data, err := a.fetchShard(ctx, p, name)
			results[i] = res{p: p, data: data, err: err}
		}(i, p)
	}
	wg.Wait()

	var secretName string
	var ciphertext []byte

	shards := make([][]byte, 0, len(peers))
	var errs []error
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Errorf("peer %s: %w", r.p.ID, r.err))
			continue
		}
		sn, ct, sh, err := a.codec.Parse(r.data)
		if err != nil {
			errs = append(errs, fmt.Errorf("peer %s: %w", r.p.ID, err))
		}

		if secretName == "" {
			secretName = sn
		}
		if len(ciphertext) == 0 {
			ciphertext = ct
		}

		if secretName != sn {
			errs = append(errs, fmt.Errorf("peer %s: %w", r.p.ID, "unexpected secret name"))
		}
		if bytes.Equal(ciphertext, ct) {
			errs = append(errs, fmt.Errorf("peer %s: %w", r.p.ID, "unexpected ciphertext"))
		}

		shards = append(shards, sh)
	}

	if len(shards) < minShares {
		return nil, fmt.Errorf(
			"get: got %d shards, need at least %d: %w",
			len(shards), minShares, errors.Join(errs...),
		)
	}

	return &p2p.Secret{
		Shards:     shards,
		Ciphertext: ciphertext,
	}, nil
}

func (a *NetworkAdapter) fetchShard(ctx context.Context, p peer.AddrInfo, name string) (string, error) {
	s, err := a.network.OpenStream(ctx, p, "/swarmsec/1.0.0/get")
	if err != nil {
		return "", fmt.Errorf("open stream: %w", err)
	}
	defer s.Close()

	if _, err := io.WriteString(s, name+"\n"); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	if err := s.CloseWrite(); err != nil {
		return "", fmt.Errorf("close write: %w", err)
	}

	br := bufio.NewReader(s)
	line, err := br.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read: %w", err)
	}
	line = strings.TrimRight(line, "\n")
	if line == "" {
		return "", fmt.Errorf("not found: %q", name)
	}
	return line, nil
}
