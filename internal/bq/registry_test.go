package bq

import (
	"testing"
)

func TestGetQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "monthly-balance-extremes",
			query:   "monthly-balance-extremes",
			wantErr: false,
		},
		{
			name:    "unknown-query",
			query:   "invalid-name",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
