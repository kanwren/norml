package main

import (
	"context"
	"fmt"
	"log"
	"os"
)

func run(_ context.Context, _ *log.Logger, _ []string) error {
	return nil
}

func main() {
	ctx := context.Background()
	logger := log.New(os.Stderr, "", log.LstdFlags)

	if err := run(ctx, logger, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
