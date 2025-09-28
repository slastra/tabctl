package main

import (
	"os"

	"github.com/tabctl/tabctl/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}