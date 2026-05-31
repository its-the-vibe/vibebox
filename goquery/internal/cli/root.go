package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/its-the-vibe/vibebox/goquery/internal/bq"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

func Run(args []string, stdout, stderr io.Writer) int {
	cmd := newRootCommand(stdout, stderr)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "goquery",
		Short:         "Run predefined BigQuery SQL queries",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.AddCommand(newQueryCommand())
	return rootCmd
}

func newQueryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "query <query-name>",
		Short: "Run a predefined BigQuery query",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
			}

			queryConfigPath := resolveQueryConfigPath()
			queryRegistry, err := bq.LoadQueryRegistry(queryConfigPath)
			if err != nil {
				return err
			}

			query, err := bq.LookupQuery(queryRegistry, args[0])
			if err != nil {
				return err
			}

			projectID := os.Getenv("GOOGLE_PROJECT_ID")
			if projectID == "" {
				return errors.New("GOOGLE_PROJECT_ID is required")
			}

			ctx := context.Background()
			service, err := bq.NewService(ctx, projectID)
			if err != nil {
				return err
			}
			defer service.Close()

			iter, err := service.Run(ctx, query)
			if err != nil {
				return err
			}

			return printRows(cmd.OutOrStdout(), iter)
		},
	}
}

func resolveQueryConfigPath() string {
	if path := os.Getenv("GOQUERY_QUERIES_FILE"); path != "" {
		return path
	}

	if executablePath, err := os.Executable(); err == nil {
		executableConfigPath := filepath.Join(filepath.Dir(executablePath), "queries.json")
		if _, err := os.Stat(executableConfigPath); err == nil {
			return executableConfigPath
		}
	}
	return "queries.json"
}

func printRows(out io.Writer, iter *bigquery.RowIterator) error {
	fmt.Fprintln(out, "Year-Month | Max Balance | Max Date   | Min Balance | Min Date")
	fmt.Fprintln(out, "----------------------------------------------------------------")

	for {
		var row []bigquery.Value
		err := iter.Next(&row)
		if errors.Is(err, iterator.Done) {
			return nil
		}
		if err != nil {
			return err
		}
		if len(row) < 5 {
			continue
		}

		fmt.Fprintf(out, "%s | %s | %s | %s | %s\n",
			formatCell(row[0]),
			formatNumber(row[1]),
			formatDate(row[2]),
			formatNumber(row[3]),
			formatDate(row[4]),
		)
	}
}

func formatCell(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func formatDate(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case civil.Date:
		return value.String()
	case time.Time:
		return value.Format("2006-01-02")
	default:
		return fmt.Sprint(v)
	}
}

func formatNumber(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case *big.Rat:
		return value.FloatString(2)
	case float64:
		return fmt.Sprintf("%.2f", value)
	case float32:
		return fmt.Sprintf("%.2f", value)
	default:
		return fmt.Sprint(v)
	}
}
