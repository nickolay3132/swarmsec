package p2pnew

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type streamsManager struct {
	host host.Host

	mu             sync.RWMutex
	streamHandlers map[protocol.ID]network.StreamHandler
}

func newStreamsManager(host host.Host) *streamsManager {
	return &streamsManager{
		host:           host,
		streamHandlers: make(map[protocol.ID]network.StreamHandler),
	}
}

func (s *streamsManager) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) error {
	if pid == protocol.ID("") {
		return fmt.Errorf("invalid stream handler protocol id")
	}
	if handler == nil {
		return fmt.Errorf("invalid stream handler")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.streamHandlers[pid]; exists {
		return fmt.Errorf("stream handler already exists")
	}

	s.streamHandlers[pid] = handler
	s.host.SetStreamHandler(pid, handler)

	return nil
}

func (s *streamsManager) RemoveStreamHandler(pid protocol.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.streamHandlers, pid)
	s.host.RemoveStreamHandler(pid)
}

func (s *streamsManager) OpenStream(ctx context.Context, ai peer.AddrInfo, pid protocol.ID) (network.Stream, error) {
	if ai.ID == s.host.ID() {
		return nil, fmt.Errorf("cannot open stream to self")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if err := s.host.Connect(ctx, ai); err != nil {
		return nil, fmt.Errorf("failed to connect to peer %s: %w", ai.ID, err)
	}

	stream, err := s.host.NewStream(ctx, ai.ID, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream to peer %s: %w", ai.ID, err)
	}

	return stream, nil
}
