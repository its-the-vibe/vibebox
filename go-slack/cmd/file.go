package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Interact with Slack files",
	Long:  `Commands for interacting with Slack files.`,
}

var fileInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get file info by file ID",
	Long:  `Retrieve information about a Slack file using its file ID.`,
	RunE:  runFileInfo,
}

var (
	fileID       string
	fileCount    int
	filePage     int
)

func init() {
	fileCmd.AddCommand(fileInfoCmd)
	fileInfoCmd.Flags().StringVarP(&fileID, "file", "f", "", "File ID (required)")
	fileInfoCmd.Flags().IntVarP(&fileCount, "count", "n", 0, "Number of comments to return per page (optional)")
	fileInfoCmd.Flags().IntVarP(&filePage, "page", "p", 0, "Page number of comments to return (optional)")
	fileInfoCmd.MarkFlagRequired("file")
}

func runFileInfo(cmd *cobra.Command, args []string) error {
	// Get Slack token from environment
	token, err := getSlackToken()
	if err != nil {
		return err
	}

	// Create Slack client
	api := slack.New(token)

	// Retrieve file info from Slack API
	file, comments, paging, err := api.GetFileInfo(fileID, fileCount, filePage)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	result := map[string]interface{}{
		"file":     file,
		"comments": comments,
		"paging":   paging,
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal file info to JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}
