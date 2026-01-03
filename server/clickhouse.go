package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func connectClickHouse() (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{os.Getenv("CLICKHOUSE_ADDR")},
		Auth: clickhouse.Auth{
			Database: os.Getenv("CLICKHOUSE_DB"),
			Username: os.Getenv("CLICKHOUSE_USER"),
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout: time.Second,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open ClickHouse connection: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return conn, nil
}

func startBatchInserter(conn clickhouse.Conn) {
	ticker := time.NewTicker(BATCH_FLUSH_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case table := <-flushSignal:
			flushTable(conn, table)
		case <-ticker.C:
			for table := range tableBuffers {
				if len(tableBuffers[table]) > 0 {
					flushTable(conn, table)
				}
			}
		case <-context.Background().Done():
			log.Println("Shutting down batch inserter...")
			for table := range tableBuffers {
				if len(tableBuffers[table]) > 0 {
					flushTable(conn, table)
				}
			}
			return
		}
	}
}

func flushTable(conn clickhouse.Conn, table string) {
	bufferMutex.Lock()
	eventsToFlush := tableBuffers[table]
	tableBuffers[table] = nil
	bufferMutex.Unlock()

	if len(eventsToFlush) == 0 {
		return
	}

	schemasMutex.RLock()
	typeSchemaInfo, schemaExists := compiledSchemas[table]
	schemasMutex.RUnlock()

	if !schemaExists {
		log.Printf("Error: Schema for table '%s' not found during flush. Events dropped.", table)
		return
	}

	columns := typeSchemaInfo.ColumnOrder

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s.%s (%s)",
		os.Getenv("CLICKHOUSE_DB"),
		table,
		strings.Join(columns, ", "),
	)

	batch, err := conn.PrepareBatch(context.Background(), insertSQL)
	if err != nil {
		log.Printf("Failed to prepare batch for table %s: %v", table, err)
		return
	}

	for _, event := range eventsToFlush {
		values := make([]any, 0, len(columns))

		for _, col := range columns {
			val, ok := event[col]
			if !ok && col != "timestamp" {
				log.Printf("Warning: Column '%s' not found in event for table %s. Skipping event.", col, table)
				continue
			}

			if !ok && col == "timestamp" {
				values = append(values, time.Now())
				continue
			}

			if col == "timestamp" {
				tsStr, isString := val.(string)
				if !isString {
					log.Printf("Warning: Timestamp for table %s is not a string. Expected string, got %T. Skipping event.", table, val)
					continue
				}

				parsedTime, err := time.Parse(time.RFC3339Nano, tsStr)
				if err != nil {
					log.Printf("Error parsing timestamp '%s' for table %s: %v. Skipping event.", tsStr, table, err)
					continue
				}

				values = append(values, parsedTime)
				continue
			}

			values = append(values, val)
		}

		if len(values) != len(columns) {
			log.Printf("Error: Mismatch in number of values for table %s. Expected %d, got %d. Skipping batch.", table, len(columns), len(values))
			return
		}

		if err := batch.Append(values...); err != nil {
			log.Printf("Failed to append event to batch for table %s: %v. Event: %+v", table, err, event)
			continue
		}
	}

	if err := batch.Send(); err != nil {
		log.Printf("Failed to send batch to ClickHouse for table %s: %v", table, err)
		return
	}
	log.Printf("Inserted %d events into ClickHouse table %s.", len(eventsToFlush), table)
}
