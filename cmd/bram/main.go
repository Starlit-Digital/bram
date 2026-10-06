package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cshaiku/bram/internal/app"
)

func main() {
	ctx := context.Background()
	if err := app.Run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "bram:", err)
		os.Exit(1)
	}
}
