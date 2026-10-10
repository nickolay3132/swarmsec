package config

type P2p struct {
	ListenAddresses []string
	BootstrapPeers  []string
}

func newP2pConfig(bootstrap string) *P2p {
	listenAddrs := "/ip4/127.0.0.1/tcp/0"

	return &P2p{
		ListenAddresses: stringToSlice(listenAddrs),
		BootstrapPeers:  stringToSlice(bootstrap),
	}
}
