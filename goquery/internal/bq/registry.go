package bq

import (
	"encoding/json"
	"fmt"
	"os"
)

type QueryRegistry map[string]string

func LoadQueryRegistry(path string) (QueryRegistry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read query config %q: %w", path, err)
	}

	var registry QueryRegistry
	if err := json.Unmarshal(content, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse query config %q: %w", path, err)
	}
	return registry, nil
}

func LookupQuery(path string, name string) (string, error) {
	queryRegistry, err := LoadQueryRegistry(path)
	if err != nil {
		return "", err
	}

	query, ok := queryRegistry[name]
	if !ok {
		return "", fmt.Errorf(`unknown query %q`, name)
	}
	return query, nil
}
