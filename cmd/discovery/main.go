package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
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

	node.RegisterProtocol("/swarmsec/put/1.0.0", func(stream network.Stream) {
		defer stream.Close()

		data, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("Read put stream: %v", err)
			return
		}

		log.Printf("Got put stream: %s", string(data))
	})

	fmt.Println("\nNode is running. Press Ctrl+C to stop.")

	input := make(chan string)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		defer close(input)

		for scanner.Scan() {
			input <- scanner.Text()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nNode is shutting down")
			return

		case line, ok := <-input:
			if !ok {
				return
			}

			switch line {
			case "message":
				fmt.Println("Sending message")
				peers, err := node.FindPeers(ctx, 3)
				if err != nil {
					if ctx.Err() == nil {
						log.Println("Find peers:", err)
					}
				} else {
					for _, p := range peers {
						fmt.Printf("[Discovered] %s | addresses: %v\n", p.ID, p.Addrs)

						stream, err := node.OpenStream(ctx, p.ID, "/swarmsec/put/1.0.0")
						if err != nil {
							log.Println("Open stream:", err)
							continue
						}

						stream.Write([]byte("hello"))

						stream.CloseWrite()
						stream.Close()
					}
				}
			}
		}
	}
	<-ctx.Done()

	fmt.Println("\nShutting down node...")

}
