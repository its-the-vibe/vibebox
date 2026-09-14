package main

import (
	"os"

	"github.com/its-the-vibe/vibebox/docker-override/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
