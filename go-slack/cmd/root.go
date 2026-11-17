package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-slack",
	Short: "A CLI tool for interacting with the Slack API",
	Long:  `go-slack is a command line tool that allows you to interact with the Slack API.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add subcommands here
	rootCmd.AddCommand(messageCmd)
	rootCmd.AddCommand(fileCmd)
}

// getSlackToken retrieves the Slack bot token from environment variable
func getSlackToken() (string, error) {
	token := os.Getenv("SLACK_BOT_TOKEN")
	if token == "" {
		return "", fmt.Errorf("SLACK_BOT_TOKEN environment variable is not set")
	}
	return token, nil
}
