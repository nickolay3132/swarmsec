package p2p

import (
	"context"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type Node struct {
	Host host.Host
	Dht  *dht.IpfsDHT
}

func NewNode(addrs []multiaddr.Multiaddr) (*Node, error) {
	h, err := libp2p.New(
		libp2p.ListenAddrs(addrs...),
	)
	if err != nil {
		return nil, err
	}

	d, err := dht.New(h,
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix("/swarmsec"),
	)
	if err != nil {
		return nil, err
	}

	return &Node{
		Host: h,
		Dht:  d,
	}, nil
}

func (n *Node) ConnectBootstrapPeers(ctx context.Context, addrs []multiaddr.Multiaddr) error {
	for _, addr := range addrs {
		addrInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}

		if err := n.Host.Connect(ctx, *addrInfo); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) Close() error {
	if err := n.Dht.Close(); err != nil {
		return err
	}

	if err := n.Host.Close(); err != nil {
		return err
	}

	return nil
}
