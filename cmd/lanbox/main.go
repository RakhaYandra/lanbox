package main

import (
	"os"

	"github.com/RakhaYandra/lanbox/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
