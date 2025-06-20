package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

type Schema struct {
	Schema      *gojsonschema.Schema
	ColumnOrder []string
}

var (
	compiledSchemas = make(map[string]Schema)
	schemasMutex    sync.RWMutex
)

func loadAndCompileSchemas(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read schemas directory %s: %w", dir, err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(file.Name(), ".json")
		path := filepath.Join(dir, file.Name())

		schemaBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", path, err)
		}

		var rawSchema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &rawSchema); err != nil {
			return fmt.Errorf("failed to unmarshal schema JSON from %s: %w", path, err)
		}

		var columnOrder []string
		if orderVal, ok := rawSchema["x-clickhouse-column-order"]; ok {
			if orderList, ok := orderVal.([]interface{}); ok {
				for _, item := range orderList {
					if colName, ok := item.(string); ok {
						columnOrder = append(columnOrder, colName)
					} else {
						return fmt.Errorf("invalid type for x-clickhouse-column-order item in %s", path)
					}
				}
			} else {
				return fmt.Errorf("x-clickhouse-column-order in %s is not an array of strings", path)
			}
		} else {
			return fmt.Errorf("x-clickhouse-column-order property missing in schema %s", path)
		}

		loader := gojsonschema.NewBytesLoader(schemaBytes)
		schema, err := gojsonschema.NewSchema(loader)
		if err != nil {
			return fmt.Errorf("failed to compile schema %s from %s: %w", name, path, err)
		}

		schemasMutex.Lock()
		compiledSchemas[name] = Schema{
			Schema:      schema,
			ColumnOrder: columnOrder,
		}
		schemasMutex.Unlock()

		log.Printf("Successfully loaded and compiled schema for table: %s, columns: %v", name, columnOrder)
	}

	return nil
}
