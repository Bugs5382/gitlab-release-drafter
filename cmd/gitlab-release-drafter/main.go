package main

import (
	"context"
	"os"
)

func main() {
	if err := newRootCmd().ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
