package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/nickolay3132/swarmsec/internal/p2p"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	var bootstrapAddr string
	flag.StringVar(&bootstrapAddr, "bootstrap", "", "Multiaddr of the bootstrap peer")
	flag.Parse()

	node, err := p2p.NewNode(ctx, p2p.Config{
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
		Bootstrap: []string{
			bootstrapAddr,
		},
		Namespace:        "swarmsec-storage",
		DiscoveryTimeout: 15 * time.Second,
		ConnectTimeout:   5 * time.Second,
		OnMessage: func(from peer.ID, data []byte) {
			fmt.Printf("\n[Message] From %s: %s\n", from, data)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer node.Close()

	fmt.Println("Node started")
	fmt.Println("Peer ID:", node.ID())
	fmt.Println("Listening addresses:")

	for _, addr := range node.Multiaddrs() {
		fmt.Println(" ", addr)
	}

	node.OnPeerConnected(func(
		id peer.ID,
		addr multiaddr.Multiaddr,
	) {
		fmt.Printf("\n[Connected] Peer: %s, address: %s\n", id, addr)
	})

	peers, err := node.FindPeers(ctx, 3)
	if err != nil {
		if ctx.Err() == nil {
			log.Println("Find peers:", err)
		}
	} else {
		for _, p := range peers {
			fmt.Printf("[Discovered] %s | addresses: %v\n", p.ID, p.Addrs)

			if err := node.Send(ctx, p.ID, []byte("hello")); err != nil {
				log.Printf("Send to %s: %v\n", p.ID, err)
			}
		}
	}

	fmt.Println("\nNode is running. Press Ctrl+C to stop.")

	<-ctx.Done()

	fmt.Println("\nShutting down node...")
	//// 6. Ждём Enter и ищем 2 пиров
	//reader := bufio.NewReader(os.Stdin)
	//for {
	//	fmt.Println("\n--- Press Enter to find 2 peers in namespace ---")
	//	_, _ = reader.ReadString('\n')
	//
	//	peerChan, err := routingDiscovery.FindPeers(ctx, namespace)
	//	if err != nil {
	//		fmt.Println("find peers:", err)
	//		continue
	//	}
	//
	//	count := 0
	//	for p := range peerChan {
	//		//if p.ID == h.ID() {
	//		//	continue // пропускаем себя
	//		//}
	//		fmt.Printf("Found peer [%d]: %s | addrs: %v\n", count+1, p.ID.String(), p.Addrs)
	//		//count++
	//		//if count >= 2 {
	//		//	break
	//		//}
	//	}
	//	if count == 0 {
	//		fmt.Println("No peers found (routing table may still be filling)")
	//	}
	//}
}
