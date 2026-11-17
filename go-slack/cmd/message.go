package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "Interact with Slack messages",
	Long:  `Commands for interacting with Slack messages.`,
}

var messageGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a message by timestamp and channel ID",
	Long:  `Retrieve a Slack message using its timestamp and channel ID.`,
	RunE:  runMessageGet,
}

var (
	channelID   string
	timestamp   string
	withReplies bool
)

func init() {
	messageCmd.AddCommand(messageGetCmd)
	messageGetCmd.Flags().StringVarP(&channelID, "channel", "c", "", "Channel ID (required)")
	messageGetCmd.Flags().StringVarP(&timestamp, "timestamp", "t", "", "Message timestamp (optional)")
	messageGetCmd.Flags().BoolVarP(&withReplies, "with-replies", "r", false, "Also retrieve replies to the message (only if timestamp is specified)")
	messageGetCmd.MarkFlagRequired("channel")
}

func runMessageGet(cmd *cobra.Command, args []string) error {
	// Get Slack token from environment
	token, err := getSlackToken()
	if err != nil {
		return err
	}

	// Create Slack client
	api := slack.New(token)

	var params *slack.GetConversationHistoryParameters
	if timestamp != "" {
		// If timestamp is specified, search for that specific message
		params = &slack.GetConversationHistoryParameters{
			ChannelID:          channelID,
			Latest:             timestamp,
			Inclusive:          true,
			Limit:              1,
			IncludeAllMetadata: true,
		}
	} else {
		// If timestamp is not specified, get the latest 100 messages
		params = &slack.GetConversationHistoryParameters{
			ChannelID:          channelID,
			Limit:              100,
			IncludeAllMetadata: true,
		}
	}

	history, err := api.GetConversationHistory(params)
	if err != nil {
		return fmt.Errorf("failed to get message(s): %w", err)
	}

	if timestamp != "" {
		// Check if we got any messages
		if len(history.Messages) == 0 {
			return fmt.Errorf("no message found with timestamp %s in channel %s", timestamp, channelID)
		}

		// Get the first message and verify it matches the requested timestamp
		message := history.Messages[0]
		if message.Timestamp != timestamp {
			return fmt.Errorf("no message found with exact timestamp %s in channel %s", timestamp, channelID)
		}

		if withReplies {
			// Fetch replies (thread)
			replies, _, _, err := api.GetConversationReplies(&slack.GetConversationRepliesParameters{
				ChannelID:          channelID,
				Timestamp:          timestamp,
				IncludeAllMetadata: true,
				Inclusive:          true,
				Limit:              100,
			})
			if err != nil {
				return fmt.Errorf("failed to get replies: %w", err)
			}
			// Output main message and replies as a JSON object
			result := map[string]interface{}{
				"message": message,
				"replies": replies,
			}
			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal message and replies to JSON: %w", err)
			}
			fmt.Println(string(jsonData))
		} else {
			// Convert message to JSON and print to stdout
			jsonData, err := json.MarshalIndent(message, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal message to JSON: %w", err)
			}
			fmt.Println(string(jsonData))
		}
	} else {
		// Print the latest 100 messages as JSON array
		if len(history.Messages) == 0 {
			return fmt.Errorf("no messages found in channel %s", channelID)
		}
		jsonData, err := json.MarshalIndent(history.Messages, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal messages to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	}
	return nil
}
