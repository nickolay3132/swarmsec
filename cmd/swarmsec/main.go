package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nickolay3132/swarmsec/internal/cli"
	"github.com/nickolay3132/swarmsec/internal/database"
	"github.com/nickolay3132/swarmsec/internal/encryption"
	"github.com/nickolay3132/swarmsec/internal/handlers"
	"github.com/nickolay3132/swarmsec/internal/p2pnew"
	"github.com/nickolay3132/swarmsec/internal/sharing"
	strs "github.com/nickolay3132/swarmsec/internal/strings"
	"github.com/nickolay3132/swarmsec/internal/usecases/secret"
	"github.com/nickolay3132/swarmsec/internal/usecases/shard"
)

func loadKeys(privKeyFile string) (database.KeyPair, error) {
	if privKeyFile == "" {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return database.KeyPair{}, fmt.Errorf("generate key: %w", err)
		}
		return database.KeyPair{Private: priv, Public: pub}, nil
	}

	kp, err := database.NewKeyStoreFS().LoadOrCreate(privKeyFile)
	if err != nil {
		return database.KeyPair{}, fmt.Errorf("load key: %w", err)
	}

	return kp, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var bootstrapAddr string
	var privKeyFile string
	flag.StringVar(&bootstrapAddr, "bootstrap", "", "Multiaddr of the bootstrap peer")
	flag.StringVar(&privKeyFile, "priv", "", "Path to private key")
	flag.Parse()

	keys, err := loadKeys(privKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	net, err := p2pnew.Init(ctx, p2pnew.Config{
		ListenAddrs:       []string{"/ip4/127.0.0.1/tcp/0"},
		BootstrapAddrs:    []string{bootstrapAddr},
		PrivateKey:        keys.Private,
		DhtProtocolPrefix: "/swarmsec",
		BootstrapTimeout:  30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	if _, err := net.AdvertiseLoop(ctx, "sws", 5*time.Second); err != nil {
		log.Fatal(err)
	}

	chiper := encryption.NewCipher()
	sharer := sharing.NewSharer()
	codec := strs.NewCodec()

	loop := cli.NewLoop(os.Stdin, os.Stdout)
	netAdapter := p2pnew.NewNetworkAdapter(net, codec)

	shardOwnersRepo := database.NewShardOwnersInMemory()
	shardsRepo := database.NewShardsInMemory()

	storeSecretUC := secret.NewStoreUC(chiper, sharer, netAdapter, shardOwnersRepo)
	retrieveSecretUC := secret.NewRetrieveUC(shardOwnersRepo, netAdapter, chiper, sharer)
	storeShardUC := shard.NewStoreUC(shardsRepo, codec)
	provideShardUC := shard.NewProvideUC(shardsRepo, codec)

	if err = net.SetStreamHandler("/swarmsec/1.0.0/put", handlers.StoreShardP2PHandler(storeShardUC)); err != nil {
		log.Fatal(err)
	}
	if err = net.SetStreamHandler("/swarmsec/1.0.0/get", handlers.RetrieveShardP2PHandler(provideShardUC)); err != nil {
		log.Fatal(err)
	}

	if err = loop.Register("addrs", handlers.PrintListenAddresses(net)); err != nil {
		log.Fatal(err)
	}
	if err = loop.Register("put", handlers.StoreShardCLIHandler(storeSecretUC)); err != nil {
		log.Fatal(err)
	}
	if err = loop.Register("get", handlers.RetrieveShardCLIHandler(retrieveSecretUC)); err != nil {
		log.Fatal(err)
	}

	if err := loop.Run(ctx); err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	stop()
}
