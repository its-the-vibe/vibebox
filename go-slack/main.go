package main

import (
	"os"

	"github.com/its-the-vibe/go-slack/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
