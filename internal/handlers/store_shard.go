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

func StoreShardCLIHandler(usecase *secret.StoreUC) cli.HandlerFunc {
	return func(ctx context.Context, args []string) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if len(args) != 2 {
			return fmt.Errorf("usage: put [NAME] [SECRET]")
		}

		if err := usecase.Execute(ctx, args[1], args[0]); err != nil {
			log.Printf("something went wrong: %v\n", err)
		}

		return nil
	}
}

func StoreShardP2PHandler(usecase *shard.StoreUC) network.StreamHandler {
	return func(s network.Stream) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		defer s.Close()

		br := bufio.NewReader(s)
		line, err := br.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Println("put handler: read error:", err)
			_, _ = io.WriteString(s, "err\n")
			return
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			_, _ = io.WriteString(s, "err\n")
			return
		}

		if err = usecase.Execute(ctx, line); err != nil {
			fmt.Println("put handler: execute error:", err)
			_, _ = io.WriteString(s, "err\n")
			return
		}

		_, _ = io.WriteString(s, "ok\n")
	}
}
