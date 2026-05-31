package bq

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
)

// Client handles BigQuery interactions.
type Client struct {
	bqClient *bigquery.Client
}

// NewClient initializes a new BigQuery client.
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	if projectID == "" {
		return nil, fmt.Errorf("GOOGLE_PROJECT_ID is missing")
	}
	bqClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create bigquery client: %w", err)
	}
	return &Client{bqClient: bqClient}, nil
}

// Close closes the underlying BigQuery client.
func (c *Client) Close() error {
	return c.bqClient.Close()
}

// ExecuteQuery executes a SQL query and returns a RowIterator.
func (c *Client) ExecuteQuery(ctx context.Context, sql string) (*bigquery.RowIterator, error) {
	q := c.bqClient.Query(sql)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	return it, nil
}
