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
	"sort"
	"strings"
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
	tablePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
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
	rootCmd.AddCommand(newListCommand())
	return rootCmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available queries",
		RunE: func(cmd *cobra.Command, args []string) error {
			queryConfigPath := resolveQueryConfigPath()
			queryRegistry, err := bq.LoadQueryRegistry(queryConfigPath)
			if err != nil {
				return err
			}

			names := make([]string, 0, len(queryRegistry))
			for name := range queryRegistry {
				names = append(names, name)
			}
			sort.Strings(names)

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Available queries:")
			for _, name := range names {
				fmt.Fprintf(out, "  - %s\n", name)
			}

			return nil
		},
	}
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
	var firstRow []bigquery.Value
	err := iter.Next(&firstRow)
	if errors.Is(err, iterator.Done) {
		return nil
	}
	if err != nil {
		return err
	}

	schema := iter.Schema

	// Collect all rows so we can compute column widths before printing.
	allValues := [][]bigquery.Value{firstRow}
	for {
		var row []bigquery.Value
		err := iter.Next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		allValues = append(allValues, row)
	}

	// Format every row as strings.
	formattedRows := make([][]string, len(allValues))
	for i, row := range allValues {
		cells := make([]string, len(row))
		for j, value := range row {
			if j < len(schema) {
				cells[j] = formatByType(value, schema[j].Type)
			} else {
				cells[j] = formatCell(value)
			}
		}
		formattedRows[i] = cells
	}

	// Build header names.
	headers := make([]string, len(schema))
	for i, field := range schema {
		headers[i] = field.Name
	}

	if len(headers) == 0 {
		for _, row := range formattedRows {
			if _, err := fmt.Fprintln(out, strings.Join(row, " | ")); err != nil {
				return err
			}
		}
		return nil
	}

	widths := calcColumnWidths(append([][]string{headers}, formattedRows...))
	sepLen := columnSepLen(widths)

	if _, err := fmt.Fprintln(out, formatTableRow(headers, widths)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, strings.Repeat("-", sepLen)); err != nil {
		return err
	}
	for _, row := range formattedRows {
		if _, err := fmt.Fprintln(out, formatTableRow(row, widths)); err != nil {
			return err
		}
	}

	return nil
}

func schemaHeader(schema bigquery.Schema) string {
	headers := make([]string, 0, len(schema))
	for _, field := range schema {
		headers = append(headers, field.Name)
	}
	return strings.Join(headers, " | ")
}

// calcColumnWidths returns the maximum character width of each column across all rows.
func calcColumnWidths(rows [][]string) []int {
	if len(rows) == 0 {
		return nil
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	return widths
}

// columnSepLen returns the total display length of a separator line for the given column widths.
func columnSepLen(widths []int) int {
	total := 0
	for i, w := range widths {
		total += w
		if i < len(widths)-1 {
			total += 3 // " | "
		}
	}
	return total
}

// formatTableRow joins cells with " | " separators, padding each to its column width.
// The last cell is not padded with trailing spaces.
func formatTableRow(cells []string, widths []int) string {
	if len(cells) == 0 {
		return ""
	}
	parts := make([]string, len(cells))
	for i, cell := range cells {
		if i < len(widths) && i != len(cells)-1 && widths[i] > len(cell) {
			parts[i] = cell + strings.Repeat(" ", widths[i]-len(cell))
		} else {
			parts[i] = cell
		}
	}
	return strings.Join(parts, " | ")
}

func printRow(out io.Writer, row []bigquery.Value, schema bigquery.Schema) error {
	cells := make([]string, len(row))
	for i, value := range row {
		if i < len(schema) {
			cells[i] = formatByType(value, schema[i].Type)
		} else {
			cells[i] = formatCell(value)
		}
	}
	_, err := fmt.Fprintln(out, strings.Join(cells, " | "))
	return err
}

func formatByType(v any, fieldType bigquery.FieldType) string {
	switch fieldType {
	case bigquery.NumericFieldType, bigquery.BigNumericFieldType, bigquery.FloatFieldType, bigquery.IntegerFieldType:
		return formatNumber(v)
	case bigquery.DateFieldType, bigquery.DateTimeFieldType, bigquery.TimestampFieldType, bigquery.TimeFieldType:
		return formatDate(v)
	default:
		return formatCell(v)
	}
}

type schemaRow struct {
	Name        string
	Type        string
	Mode        string
	Description string
}

func printSchemaRows(out io.Writer, iter *bigquery.RowIterator, dataset, table string) error {
	// Collect all rows so we can compute column widths before printing.
	var rows []schemaRow
	for {
		var row schemaRow
		err := iter.Next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return fmt.Errorf("no schema information found for table %q in dataset %q", table, dataset)
	}

	headers := []string{"Name", "Type", "Mode", "Description"}

	formattedRows := make([][]string, len(rows))
	for i, row := range rows {
		formattedRows[i] = []string{
			formatCell(row.Name),
			formatCell(row.Type),
			formatCell(row.Mode),
			formatCell(row.Description),
		}
	}

	widths := calcColumnWidths(append([][]string{headers}, formattedRows...))
	sepLen := columnSepLen(widths)

	if _, err := fmt.Fprintln(out, formatTableRow(headers, widths)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, strings.Repeat("-", sepLen)); err != nil {
		return err
	}
	for _, row := range formattedRows {
		if _, err := fmt.Fprintln(out, formatTableRow(row, widths)); err != nil {
			return err
		}
	}

	return nil
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
