package p2p

import (
	"context"
	"errors"
	"fmt"
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

	AdvertiseInterval time.Duration
	DiscoveryTimeout  time.Duration
	ConnectTimeout    time.Duration
}

type StreamHandler func(network.Stream)

type Node struct {
	host      host.Host
	dht       *dht.IpfsDHT
	discovery *drouting.RoutingDiscovery

	namespace string
	config    Config

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	protocolMu       sync.RWMutex
	protocolHandlers map[protocol.ID]StreamHandler

	findMu     sync.Mutex
	lastResult map[peer.ID]struct{}

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
		host:             h,
		namespace:        cfg.Namespace,
		config:           cfg,
		ctx:              ctx,
		cancel:           cancel,
		protocolHandlers: make(map[protocol.ID]StreamHandler),
		lastResult:       make(map[peer.ID]struct{}),
	}

	fail := func(err error) (*Node, error) {
		cancel()

		if n.dht != nil {
			_ = n.dht.Close()
		}

		_ = h.Close()
		return nil, err
	}

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
			ctx,
			cfg.ConnectTimeout,
		)

		err = h.Connect(connectCtx, *info)
		connectCancel()

		if err != nil {
			return fail(fmt.Errorf(
				"connect bootstrap %s: %w",
				info.ID,
				err,
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

func (n *Node) Host() host.Host {
	return n.host
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

func (n *Node) RegisterProtocol(id protocol.ID, handler StreamHandler) error {
	if id == "" {
		return errors.New("protocol ID cannot be empty")
	}
	if handler == nil {
		return errors.New("stream handler cannot be nil")
	}

	n.protocolMu.Lock()
	defer n.protocolMu.Unlock()

	if n.ctx.Err() != nil {
		return errors.New("node is closed")
	}

	if _, exists := n.protocolHandlers[id]; exists {
		return fmt.Errorf("protocol %q is already registered", id)
	}

	n.protocolHandlers[id] = handler

	n.host.SetStreamHandler(id, func(stream network.Stream) {
		defer stream.Close()

		handler(stream)
	})

	return nil
}

func (n *Node) UnregisterProtocol(id protocol.ID) {
	n.protocolMu.Lock()
	defer n.protocolMu.Unlock()

	delete(n.protocolHandlers, id)
	n.host.RemoveStreamHandler(id)
}

func (n *Node) OpenStream(ctx context.Context, id peer.ID, proto protocol.ID) (network.Stream, error) {
	if id == n.ID() {
		return nil, errors.New("cannot open stream to self")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if n.ctx.Err() != nil {
		return nil, errors.New("node is closed")
	}

	stream, err := n.host.NewStream(ctx, id, proto)
	if err != nil {
		return nil, fmt.Errorf("open stream to %s using %s: %w", id, proto, err)
	}

	return stream, nil
}

func (n *Node) Connect(ctx context.Context, p peer.AddrInfo) error {
	if p.ID == n.ID() {
		return errors.New("cannot connect to self")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if n.ctx.Err() != nil {
		return errors.New("node is closed")
	}

	connectCtx, cancel := context.WithTimeout(
		ctx,
		n.config.ConnectTimeout,
	)
	defer cancel()

	if err := n.host.Connect(connectCtx, p); err != nil {
		return fmt.Errorf("connect to %s: %w", p.ID, err)
	}

	return nil
}

func (n *Node) FindPeers(ctx context.Context, count int) ([]peer.AddrInfo, error) {
	if count <= 0 {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if n.ctx.Err() != nil {
		return nil, errors.New("node is closed")
	}

	n.findMu.Lock()
	defer n.findMu.Unlock()

	findCtx, cancel := context.WithTimeout(
		ctx,
		n.config.DiscoveryTimeout,
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
			return nil, fmt.Errorf("could not find enough peers (found %d/%d): %w", len(result), count, findCtx.Err())

		case p, ok := <-ch:
			if !ok {
				n.lastResult = current
				return nil, fmt.Errorf("peer discovery channel closed prematurely (found %d/%d)", len(result), count)
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

			if _, exists := n.lastResult[p.ID]; exists {
				continue
			}

			if err := n.Connect(ctx, p); err != nil {
				continue
			}

			current[p.ID] = struct{}{}
			result = append(result, p)
		}
	}

	n.lastResult = current
	return result, nil
}

func (n *Node) advertise() {
	dutil.Advertise(n.ctx, n.discovery, n.namespace)
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
		n.wg.Wait()

		var dhtErr error
		if n.dht != nil {
			dhtErr = n.dht.Close()
		}

		hostErr := n.host.Close()
		n.closeErr = errors.Join(dhtErr, hostErr)
	})

	return n.closeErr
}
