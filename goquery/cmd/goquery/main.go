package main

import (
	"os"

	"github.com/its-the-vibe/vibebox/goquery/internal/cli"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
