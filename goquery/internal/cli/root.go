package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/its-the-vibe/vibebox/goquery/internal/bq"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

var (
	projectIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{5,29}$`)
	datasetPattern   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	tablePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
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
	rootCmd.AddCommand(newSchemaCommand())
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

func newSchemaCommand() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "schema <dataset> <table>",
		Short: "Inspect a BigQuery table schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) != 2 {
				return fmt.Errorf("accepts 2 arg(s), received %d", len(args))
			}

			dataset := args[0]
			table := args[1]
			if !isValidDatasetName(dataset) {
				return fmt.Errorf("invalid dataset name %q", dataset)
			}
			if !isValidTableName(table) {
				return fmt.Errorf("invalid table name %q", table)
			}

			if projectID == "" {
				projectID = os.Getenv("GOOGLE_PROJECT_ID")
			}
			if projectID == "" {
				return errors.New("GOOGLE_PROJECT_ID is required")
			}
			if !isValidProjectID(projectID) {
				return fmt.Errorf("invalid project id %q", projectID)
			}

			ctx := context.Background()
			service, err := bq.NewService(ctx, projectID)
			if err != nil {
				return err
			}
			defer service.Close()

			schemaQuery := fmt.Sprintf("SELECT\n"+
				"  c.column_name AS name,\n"+
				"  c.data_type AS type,\n"+
				"  CASE\n"+
				"    WHEN STARTS_WITH(c.data_type, 'ARRAY<') THEN 'REPEATED'\n"+
				"    WHEN c.is_nullable = 'YES' THEN 'NULLABLE'\n"+
				"    ELSE 'REQUIRED'\n"+
				"  END AS mode,\n"+
				"  COALESCE(cfp.description, '') AS description\n"+
				"FROM\n"+
				"  `%s.%s.INFORMATION_SCHEMA.COLUMNS` c\n"+
				"LEFT JOIN\n"+
				"  `%s.%s.INFORMATION_SCHEMA.COLUMN_FIELD_PATHS` cfp\n"+
				"ON\n"+
				"  c.table_name = cfp.table_name\n"+
				"  AND c.column_name = cfp.column_name\n"+
				"WHERE\n"+
				"  c.table_name = @table_name\n"+
				"ORDER BY\n"+
				"  c.ordinal_position", projectID, dataset, projectID, dataset)

			iter, err := service.RunWithParameters(ctx, schemaQuery, []bigquery.QueryParameter{
				{Name: "table_name", Value: table},
			})
			if err != nil {
				return err
			}

			return printSchemaRows(cmd.OutOrStdout(), iter, dataset, table)
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Google Cloud project ID (defaults to GOOGLE_PROJECT_ID)")
	return cmd
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

type schemaRow struct {
	Name        string
	Type        string
	Mode        string
	Description string
}

func printSchemaRows(out io.Writer, iter *bigquery.RowIterator, dataset, table string) error {
	fmt.Fprintln(out, "Name | Type | Mode | Description")
	fmt.Fprintln(out, "--------------------------------")

	found := false
	for {
		var row schemaRow
		err := iter.Next(&row)
		if errors.Is(err, iterator.Done) {
			if !found {
				return fmt.Errorf("no schema information found for table %q in dataset %q", table, dataset)
			}
			return nil
		}
		if err != nil {
			return err
		}

		found = true
		fmt.Fprintf(out, "%s | %s | %s | %s\n",
			formatCell(row.Name),
			formatCell(row.Type),
			formatCell(row.Mode),
			formatCell(row.Description),
		)
	}
}

func isValidProjectID(projectID string) bool {
	return projectIDPattern.MatchString(projectID)
}

func isValidDatasetName(dataset string) bool {
	return len(dataset) >= 1 && len(dataset) <= 1024 && datasetPattern.MatchString(dataset)
}

func isValidTableName(table string) bool {
	return len(table) >= 1 && len(table) <= 1024 && tablePattern.MatchString(table)
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
