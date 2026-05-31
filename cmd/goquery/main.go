package main

import (
	"fmt"
	"os"

	"github.com/its-the-vibe/vibebox/goquery/internal/cli"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "query") {
		if os.Getenv("GOOGLE_PROJECT_ID") == "" {
			fmt.Fprintln(os.Stderr, "Error: GOOGLE_PROJECT_ID environment variable is not set")
			os.Exit(1)
		}
	}

	cli.Execute()
}
