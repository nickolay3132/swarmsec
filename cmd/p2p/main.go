package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/multiformats/go-multiaddr"
	"github.com/nickolay3132/swarmsec/internal/p2p"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var bootstrapAddr string
	flag.StringVar(&bootstrapAddr, "bootstrap", "", "Multiaddr of the bootstrap peer")
	flag.Parse()

	addr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	if err != nil {
		log.Fatal(err)
	}

	node, err := p2p.NewNode([]multiaddr.Multiaddr{addr})
	if err != nil {
		log.Fatal(err)
	}
	defer node.Close()

	fmt.Println("Addresses:")
	for _, addr := range node.Host.Addrs() {
		fmt.Printf("%s/p2p/%s\n", addr, node.Host.ID().String())
	}

	if bootstrapAddr != "" {
		bootstrapMA, err := multiaddr.NewMultiaddr(bootstrapAddr)
		if err != nil {
			log.Fatal(err)
		}

		if err := node.ConnectBootstrapPeers(ctx, []multiaddr.Multiaddr{bootstrapMA}); err != nil {
			log.Fatal(err)
		}
	}

	if err := node.Dht.Bootstrap(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nPress Enter to see DHT table")

	reader := bufio.NewReader(os.Stdin)

	for {
		_, _ = reader.ReadString('\n')

		fmt.Println("\n--- ТЕКУЩИЕ ПИРЫ В DHT ТАБЛИЦЕ МАРШРУТИЗАЦИИ ---")

		// Получаем список всех ID пиров, о которых знает DHT
		routingTablePeers := node.Dht.RoutingTable().ListPeers()

		if len(routingTablePeers) == 0 {
			fmt.Println("Table is empty")
		} else {
			for i, peerID := range routingTablePeers {
				// Пытаемся узнать адреса пира из локального хранилища (Peerstore)
				addrs := node.Host.Peerstore().Addrs(peerID)
				fmt.Printf("[%d] ID: %s | Address: %v\n", i+1, peerID.String(), addrs)
			}
		}
		fmt.Println("------------------------------------------------\n")
	}
}
