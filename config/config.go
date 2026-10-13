package config

import (
	"flag"
	"strings"
)

type Config struct {
	P2p *P2p
}

func LoadConfig() *Config {
	var (
		bootstrap = flag.String("bootstrap", "", "Multiaddr of the bootstrap peer")
	)

	flag.Parse()

	return &Config{
		P2p: newP2pConfig(*bootstrap),
	}
}

func stringToSlice(s string) []string {
	slice := strings.Split(s, ",")
	for i := range slice {
		slice[i] = strings.TrimSpace(slice[i])
	}
	return slice
}
