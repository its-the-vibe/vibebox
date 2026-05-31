package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goquery",
	Short: "goquery is a CLI tool to run predefined BigQuery queries",
	Long:  `A command-line utility named goquery to run predefined SQL queries against Google Cloud BigQuery.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(queryCmd)
}
