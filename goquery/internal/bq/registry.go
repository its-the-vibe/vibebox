package bq

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type QueryRegistry map[string]string

type queryDefinition struct {
	SQL  string `json:"sql"`
	File string `json:"file"`
}

func LoadQueryRegistry(path string) (QueryRegistry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read query config %q: %w", path, err)
	}

	var definitions map[string]queryDefinition
	if err := json.Unmarshal(content, &definitions); err != nil {
		return nil, fmt.Errorf("failed to parse query config %q: %w", path, err)
	}

	registry := make(QueryRegistry, len(definitions))
	baseDir := filepath.Dir(path)
	for name, definition := range definitions {
		switch {
		case definition.SQL != "":
			registry[name] = definition.SQL
		case definition.File != "":
			sqlPath := filepath.Join(baseDir, definition.File)
			sqlContent, err := os.ReadFile(sqlPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read SQL file %q for query %q: %w", sqlPath, name, err)
			}
			registry[name] = string(sqlContent)
		default:
			return nil, fmt.Errorf("query %q has no sql or file value", name)
		}
	}

	return registry, nil
}

func LookupQuery(queryRegistry QueryRegistry, name string) (string, error) {
	query, ok := queryRegistry[name]
	if !ok {
		return "", fmt.Errorf(`unknown query %q`, name)
	}
	return query, nil
}
