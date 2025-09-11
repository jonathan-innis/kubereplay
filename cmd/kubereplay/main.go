package main

import (
	"os"

	"github.com/joinnis/kubereplay/pkg/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
