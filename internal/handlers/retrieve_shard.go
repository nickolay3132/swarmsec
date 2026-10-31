package handlers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/nickolay3132/swarmsec/internal/cli"
	"github.com/nickolay3132/swarmsec/internal/usecases/secret"
	"github.com/nickolay3132/swarmsec/internal/usecases/shard"
)

func RetrieveShardCLIHandler(usecase *secret.RetrieveUC) cli.HandlerFunc {
	return func(ctx context.Context, args []string) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if len(args) != 1 {
			return fmt.Errorf("usage: get [NAME]")
		}

		res, err := usecase.Execute(ctx, args[0])
		if err != nil {
			log.Printf("something went wrong: %v\n", err)
		}

		log.Printf("Result: %s", string(res))
		return nil
	}
}

func RetrieveShardP2PHandler(usecase *shard.ProvideUC) network.StreamHandler {
	return func(s network.Stream) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		defer s.Close()

		br := bufio.NewReader(s)
		name, err := br.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Println("get handler: read error:", err)
			return
		}
		name = strings.TrimRight(name, "\n")
		if name == "" {
			return
		}

		secret, err := usecase.Execute(ctx, name)
		if err != nil {
			fmt.Println("get handler: execute error:", err)
			return
		}

		if _, err := io.WriteString(s, secret+"\n"); err != nil {
			fmt.Println("get handler: write error:", err)
		}
	}
}
