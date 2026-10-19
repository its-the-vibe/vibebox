package bq

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type QueryRegistry map[string]string

// ParameterDefinition declares a named, typed query parameter.
type ParameterDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ParameterRegistry maps query names to their declared parameters.
type ParameterRegistry map[string][]ParameterDefinition

type queryDefinition struct {
	SQL        string                `json:"sql"`
	File       string                `json:"file"`
	Parameters []ParameterDefinition `json:"parameters"`
}

func LoadQueryRegistry(path string) (QueryRegistry, error) {
	registry, _, err := LoadQueryConfig(path)
	return registry, err
}

// LoadQueryConfig loads the queries and their declared parameters.
func LoadQueryConfig(path string) (QueryRegistry, ParameterRegistry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read query config %q: %w", path, err)
	}

	var definitions map[string]queryDefinition
	if err := json.Unmarshal(content, &definitions); err != nil {
		return nil, nil, fmt.Errorf("failed to parse query config %q: %w", path, err)
	}

	registry := make(QueryRegistry, len(definitions))
	parameters := make(ParameterRegistry, len(definitions))
	baseDir := filepath.Dir(path)
	for name, definition := range definitions {
		for _, parameter := range definition.Parameters {
			if parameter.Name == "" || parameter.Type == "" {
				return nil, nil, fmt.Errorf("query %q has a parameter without name or type", name)
			}
		}
		parameters[name] = definition.Parameters
		switch {
		case definition.SQL != "":
			registry[name] = definition.SQL
		case definition.File != "":
			sqlPath := filepath.Join(baseDir, definition.File)
			sqlContent, err := os.ReadFile(sqlPath)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read SQL file %q for query %q: %w", sqlPath, name, err)
			}
			registry[name] = string(sqlContent)
		default:
			return nil, nil, fmt.Errorf("query %q has no sql or file value", name)
		}
	}

	return registry, parameters, nil
}

func LookupQuery(queryRegistry QueryRegistry, name string) (string, error) {
	query, ok := queryRegistry[name]
	if !ok {
		return "", fmt.Errorf(`unknown query %q`, name)
	}
	return query, nil
}
