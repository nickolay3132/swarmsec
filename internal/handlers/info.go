package handlers

import (
	"context"
	"fmt"

	"github.com/nickolay3132/swarmsec/internal/cli"
	"github.com/nickolay3132/swarmsec/internal/p2pnew"
)

func PrintListenAddresses(net *p2pnew.Network) cli.HandlerFunc {
	return func(ctx context.Context, args []string) error {
		fmt.Println("Listening addresses:")
		for _, l := range net.ListenAddresses() {
			if l == "" {
				continue
			}
			fmt.Println("\t", l)
		}
		return nil
	}
}
