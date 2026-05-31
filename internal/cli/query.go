package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"cloud.google.com/go/bigquery"
	"github.com/its-the-vibe/vibebox/goquery/internal/bq"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

var queryCmd = &cobra.Command{
	Use:   "query [query-name]",
	Short: "Run a predefined BigQuery query",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		queryName := args[0]
		sql, err := bq.GetQuery(queryName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		projectID := os.Getenv("GOOGLE_PROJECT_ID")
		if projectID == "" {
			fmt.Fprintln(os.Stderr, "Error: GOOGLE_PROJECT_ID environment variable is not set")
			os.Exit(1)
		}

		ctx := context.Background()
		client, err := bq.NewClient(ctx, projectID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		it, err := client.ExecuteQuery(ctx, sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		if queryName == "monthly-balance-extremes" {
			fmt.Fprintln(w, "Year-Month\t| Max Balance\t| Max Date\t| Min Balance\t| Min Date")
			fmt.Fprintln(w, "----------------------------------------------------------------")
		}

		for {
			var values []bigquery.Value
			err := it.Next(&values)
			if err == iterator.Done {
				break
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching rows: %v\n", err)
				os.Exit(1)
			}

			for i, val := range values {
				if f, ok := val.(float64); ok && queryName == "monthly-balance-extremes" && (i == 1 || i == 3) {
					fmt.Fprintf(w, "%.2f", f)
				} else {
					fmt.Fprint(w, val)
				}

				if i < len(values)-1 {
					fmt.Fprint(w, "\t| ")
				}
			}
			fmt.Fprintln(w)
		}
		w.Flush()
	},
}
