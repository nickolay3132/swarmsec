package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

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
