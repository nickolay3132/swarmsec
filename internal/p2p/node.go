package p2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

type Config struct {
	ListenAddrs []string
	Bootstrap   []string

	Namespace string
	Protocol  protocol.ID

	AdvertiseInterval time.Duration
	DiscoveryTimeout  time.Duration
	ConnectTimeout    time.Duration

	OnMessage func(peer.ID, []byte)
}

type PeerConnectedHandler func(peer.ID, multiaddr.Multiaddr)

type Node struct {
	host      host.Host
	dht       *dht.IpfsDHT
	discovery *drouting.RoutingDiscovery

	namespace string
	protocol  protocol.ID
	config    Config

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu             sync.RWMutex
	connectedHooks []PeerConnectedHandler
	onMessage      func(peer.ID, []byte)

	findMu     sync.Mutex
	lastResult map[peer.ID]struct{}

	notifyBundle *network.NotifyBundle

	closeOnce sync.Once
	closeErr  error
}

func NewNode(parent context.Context, cfg Config) (*Node, error) {
	if parent == nil {
		parent = context.Background()
	}
	if len(cfg.ListenAddrs) == 0 {
		cfg.ListenAddrs = []string{"/ip4/0.0.0.0/tcp/0"}
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "swarmsec-storage"
	}
	if cfg.Protocol == "" {
		cfg.Protocol = protocol.ID("/swarmsec/1.0.0")
	}
	if cfg.AdvertiseInterval <= 0 {
		cfg.AdvertiseInterval = 20 * time.Minute
	}
	if cfg.DiscoveryTimeout <= 0 {
		cfg.DiscoveryTimeout = 15 * time.Second
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(parent)

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(cfg.ListenAddrs...),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create host: %w", err)
	}

	n := &Node{
		host:       h,
		namespace:  cfg.Namespace,
		protocol:   cfg.Protocol,
		config:     cfg,
		ctx:        ctx,
		cancel:     cancel,
		onMessage:  cfg.OnMessage,
		lastResult: make(map[peer.ID]struct{}),
	}

	fail := func(err error) (*Node, error) {
		cancel()
		if n.dht != nil {
			_ = n.dht.Close()
		}
		_ = h.Close()
		return nil, err
	}

	h.SetStreamHandler(n.protocol, n.handleStream)

	n.notifyBundle = &network.NotifyBundle{
		ConnectedF: func(_ network.Network, conn network.Conn) {
			n.mu.RLock()
			hooks := append(
				[]PeerConnectedHandler(nil),
				n.connectedHooks...,
			)
			n.mu.RUnlock()

			for _, hook := range hooks {
				hook(conn.RemotePeer(), conn.RemoteMultiaddr())
			}
		},
	}
	h.Network().Notify(n.notifyBundle)

	kdht, err := dht.New(
		h,
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix("/swarmsec"),
	)
	if err != nil {
		return fail(fmt.Errorf("create DHT: %w", err))
	}
	n.dht = kdht
	n.discovery = drouting.NewRoutingDiscovery(kdht)

	for _, addr := range cfg.Bootstrap {
		if addr == "" {
			continue
		}

		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return fail(fmt.Errorf("parse bootstrap address: %w", err))
		}

		info, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return fail(fmt.Errorf("parse bootstrap peer: %w", err))
		}

		connectCtx, connectCancel := context.WithTimeout(
			ctx, cfg.ConnectTimeout,
		)
		err = h.Connect(connectCtx, *info)
		connectCancel()

		if err != nil {
			return fail(fmt.Errorf(
				"connect bootstrap %s: %w", info.ID, err,
			))
		}
	}

	if err := kdht.Bootstrap(ctx); err != nil {
		return fail(fmt.Errorf("bootstrap DHT: %w", err))
	}

	n.advertise()

	n.wg.Add(1)
	go n.advertiseLoop()

	return n, nil
}

func (n *Node) ID() peer.ID {
	return n.host.ID()
}

func (n *Node) Addrs() []multiaddr.Multiaddr {
	return n.host.Addrs()
}

func (n *Node) Multiaddrs() []string {
	addrs := n.host.Addrs()
	result := make([]string, 0, len(addrs))

	for _, addr := range addrs {
		result = append(result,
			fmt.Sprintf("%s/p2p/%s", addr, n.ID()))
	}

	return result
}

func (n *Node) OnPeerConnected(h PeerConnectedHandler) {
	if h == nil {
		return
	}

	n.mu.Lock()
	n.connectedHooks = append(n.connectedHooks, h)
	n.mu.Unlock()
}

func (n *Node) SetOnMessage(h func(peer.ID, []byte)) {
	n.mu.Lock()
	n.onMessage = h
	n.mu.Unlock()
}

// FindPeers returns up to count unique peers that were successfully
// connected to during this call.
//
// Peers returned by the immediately preceding successful search are
// skipped when possible. Older results may appear again.
func (n *Node) FindPeers(
	ctx context.Context,
	count int,
) ([]peer.AddrInfo, error) {
	if count <= 0 {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Ensures two concurrent calls cannot use the same previous result.
	n.findMu.Lock()
	defer n.findMu.Unlock()

	findCtx, cancel := context.WithTimeout(
		ctx, n.config.DiscoveryTimeout,
	)
	defer cancel()

	ch, err := n.discovery.FindPeers(findCtx, n.namespace)
	if err != nil {
		return nil, fmt.Errorf("discover peers: %w", err)
	}

	result := make([]peer.AddrInfo, 0, count)
	current := make(map[peer.ID]struct{}, count)
	attempted := make(map[peer.ID]struct{})

	for len(result) < count {
		select {
		case <-findCtx.Done():
			n.lastResult = current

			// Return partial results if we found any peers.
			if len(result) > 0 {
				return result, nil
			}
			return nil, findCtx.Err()

		case p, ok := <-ch:
			if !ok {
				n.lastResult = current
				return result, nil
			}

			if p.ID == "" || p.ID == n.ID() {
				continue
			}
			if _, exists := current[p.ID]; exists {
				continue
			}
			if _, exists := attempted[p.ID]; exists {
				continue
			}
			attempted[p.ID] = struct{}{}

			// Prefer different peers from the previous result.
			if _, exists := n.lastResult[p.ID]; exists {
				continue
			}

			connectCtx, connectCancel := context.WithTimeout(
				findCtx, n.config.ConnectTimeout,
			)
			err := n.host.Connect(connectCtx, p)
			connectCancel()

			if err != nil {
				continue
			}

			current[p.ID] = struct{}{}
			result = append(result, p)
		}
	}

	n.lastResult = current
	return result, nil
}

func (n *Node) Connect(
	ctx context.Context,
	p peer.AddrInfo,
) error {
	if p.ID == n.ID() {
		return errors.New("cannot connect to self")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	connectCtx, cancel := context.WithTimeout(
		ctx, n.config.ConnectTimeout,
	)
	defer cancel()

	if err := n.host.Connect(connectCtx, p); err != nil {
		return fmt.Errorf("connect to %s: %w", p.ID, err)
	}
	return nil
}

func (n *Node) Send(
	ctx context.Context,
	id peer.ID,
	data []byte,
) error {
	if id == n.ID() {
		return errors.New("cannot send to self")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	stream, err := n.host.NewStream(ctx, id, n.protocol)
	if err != nil {
		return fmt.Errorf("open stream to %s: %w", id, err)
	}
	defer stream.Close()

	if _, err := stream.Write(data); err != nil {
		_ = stream.Reset()
		return fmt.Errorf("write to %s: %w", id, err)
	}

	if err := stream.CloseWrite(); err != nil {
		return fmt.Errorf("close write to %s: %w", id, err)
	}
	return nil
}

func (n *Node) handleStream(stream network.Stream) {
	defer stream.Close()

	const maxMessageSize = 1 << 20 // 1 MiB

	data, err := io.ReadAll(
		io.LimitReader(stream, maxMessageSize+1),
	)
	if err != nil {
		_ = stream.Reset()
		return
	}
	if len(data) > maxMessageSize {
		_ = stream.Reset()
		return
	}

	n.mu.RLock()
	handler := n.onMessage
	n.mu.RUnlock()

	if handler != nil {
		handler(stream.Conn().RemotePeer(), data)
	}
}

func (n *Node) advertise() {
	dutil.Advertise(
		n.ctx, n.discovery, n.namespace,
	)
}

func (n *Node) advertiseLoop() {
	defer n.wg.Done()

	ticker := time.NewTicker(n.config.AdvertiseInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.advertise()
		}
	}
}

func (n *Node) Close() error {
	n.closeOnce.Do(func() {
		n.cancel()
		n.host.Network().StopNotify(n.notifyBundle)
		n.wg.Wait()

		dhtErr := n.dht.Close()
		hostErr := n.host.Close()
		n.closeErr = errors.Join(dhtErr, hostErr)
	})

	return n.closeErr
}
