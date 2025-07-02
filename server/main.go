package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/xeipuuv/gojsonschema"
)

const (
	MAX_BATCH_SIZE       = 1000
	BATCH_FLUSH_INTERVAL = 10 * time.Second
	SCHEMA_DIR           = "./schemas"
)

var (
	tableBuffers = make(map[string][]map[string]interface{})
	bufferMutex  sync.Mutex
	flushSignal  = make(chan string, 100)
	message, _   = os.ReadFile("message.txt")
	apiToken     = os.Getenv("API_TOKEN")
)

func main() {
	if err := loadAndCompileSchemas(SCHEMA_DIR); err != nil {
		log.Fatalf("Failed to load and compile JSON schemas: %v", err)
	}

	conn, err := connectClickHouse()
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}

	defer conn.Close()

	go startBatchInserter(conn)

	http.HandleFunc("GET /", get)
	http.HandleFunc("POST /api/register/{table}", register)
	http.HandleFunc("GET /api/count/{table}", func(w http.ResponseWriter, r *http.Request) {
		count(w, r, conn)
	})

	log.Printf("ready!")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func get(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(message)
}

func register(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	if table == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if r.Header.Get("Authorization") != "Bearer "+apiToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	schemasMutex.RLock()
	schema, exists := compiledSchemas[table]
	schemasMutex.RUnlock()

	if !exists {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var eventData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &eventData); err != nil {
		http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	loader := gojsonschema.NewBytesLoader(bodyBytes)
	result, err := schema.Schema.Validate(loader)
	if err != nil {
		log.Printf("Schema validation internal error for table %s: %v", table, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !result.Valid() {
		validationErrors := make([]string, 0)
		for _, desc := range result.Errors() {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: %s", desc.Field(), desc.Description()))
		}
		log.Printf("Invalid event payload for table %s: %s", table, strings.Join(validationErrors, "; "))
		http.Error(w, fmt.Sprintf("Invalid event payload: %s", strings.Join(validationErrors, "; ")), http.StatusBadRequest)
		return
	}

	bufferMutex.Lock()
	tableBuffers[table] = append(tableBuffers[table], eventData)
	currentBufferLen := len(tableBuffers[table])
	bufferMutex.Unlock()

	if currentBufferLen >= MAX_BATCH_SIZE {
		select {
		case flushSignal <- table:
		default:
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func count(w http.ResponseWriter, r *http.Request, conn clickhouse.Conn) {
	table := r.PathValue("table")
	if table == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if r.Header.Get("Authorization") != "Bearer "+apiToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	schemasMutex.RLock()
	_, exists := compiledSchemas[table]
	schemasMutex.RUnlock()

	if !exists {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	selectSQL := fmt.Sprintf(
		"SELECT name, type, count() AS total_count FROM %s.%s GROUP BY name, type",
		os.Getenv("CLICKHOUSE_DB"),
		table,
	)

	var count []struct {
		Name       string `ch:"name" json:"name"`
		Type       string `ch:"type" json:"type"`
		TotalCount uint64 `ch:"total_count" json:"count"`
	}

	err := conn.Select(context.Background(), &count, selectSQL)
	if err != nil {
		log.Printf("Error executing SELECT query for table %s: %v", table, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(count)
}
