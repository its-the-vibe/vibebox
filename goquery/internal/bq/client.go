package bq

import (
	"context"

	"cloud.google.com/go/bigquery"
)

type Service struct {
	client *bigquery.Client
}

func NewService(ctx context.Context, projectID string) (*Service, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &Service{client: client}, nil
}

func (s *Service) Close() error {
	return s.client.Close()
}

func (s *Service) Run(ctx context.Context, query string) (*bigquery.RowIterator, error) {
	return s.client.Query(query).Read(ctx)
}
