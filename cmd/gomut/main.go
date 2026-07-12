package main

import (
	"context"
	"fmt"
	"os"

	"gomut/internal/gomut"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cmd := gomut.NewCommand(os.Stdout, os.Stderr)
	return cmd.Run(ctx, args)
}

