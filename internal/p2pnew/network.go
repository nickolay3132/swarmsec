package p2pnew

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
)

type Config struct {
	ListenAddrs        []string
	BootstrapAddrs     []string
	DhtProtocolPrefix  string
	AdvertiseNamespace string

	BootstrapTimeout time.Duration
}

type Network struct {
	host      host.Host
	dht       *dht.IpfsDHT
	discovery *drouting.RoutingDiscovery

	cancel context.CancelFunc

	*ContentManager
	*RendezvousManager
	*StreamsManager
}

func Init(ctx context.Context, cfg Config) (*Network, error) {
	ctx, cancel := context.WithCancel(ctx)

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(cfg.ListenAddrs...))
	if err != nil {
		return nil, err
	}

	n := &Network{
		host:   h,
		cancel: cancel,
	}

	fail := func(err error) (*Network, error) {
		cancel()

		if n.dht != nil {
			_ = n.dht.Close()
		}

		if n.host != nil {
			_ = n.host.Close()
		}

		return nil, err
	}

	kdht, err := dht.New(h,
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(protocol.ID(cfg.DhtProtocolPrefix)))
	if err != nil {
		return fail(err)
	}

	n.dht = kdht
	n.discovery = drouting.NewRoutingDiscovery(kdht)

	if err := n.bootstrap(ctx,
		cfg.BootstrapAddrs,
		cfg.BootstrapTimeout); err != nil {
		return fail(err)
	}

	n.ContentManager = NewContentManager(kdht)
	n.RendezvousManager = NewRendezvousManager(n.host.ID(), n.discovery)
	n.StreamsManager = NewStreamsManager(n.host)

	return n, nil
}

func (n *Network) ListenAddresses() []string {
	peerID := n.host.ID()

	addrs := make([]string, len(n.host.Addrs()))

	for _, addr := range n.host.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), peerID)
		addrs = append(addrs, fullAddr)
	}

	return addrs
}

func (n *Network) Close() error {
	if n.cancel != nil {
		n.cancel()
	}

	var errs []error

	if n.dht != nil {
		if err := n.dht.Close(); err != nil {
			errs = append(errs, fmt.Errorf("dht close: %w", err))
		}
	}
	if n.host != nil {
		if err := n.host.Close(); err != nil {
			errs = append(errs, fmt.Errorf("host close: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (n *Network) bootstrap(ctx context.Context, addrs []string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addrs = slices.DeleteFunc(addrs, func(s string) bool {
		return s == ""
	})

	if len(addrs) != 0 {
		var connected int

		for _, addr := range addrs {
			if err := ctx.Err(); err != nil {
				break
			}

			ai, err := strToAddrInfo(addr)
			if err != nil {
				// TODO: Add logging
				continue
			}

			if err := n.host.Connect(ctx, *ai); err != nil {
				// TODO: Add logging
				continue
			}
			connected++
		}

		if connected == 0 {
			return errors.New("p2p bootstrap: no reachable addresses")
		}
	}

	return n.dht.Bootstrap(ctx)
}

func strToAddrInfo(addr string) (*peer.AddrInfo, error) {
	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}

	info, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return nil, err
	}

	return info, nil
}
