package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// JSONLWriter streams extracted records line-by-line to a file.
type JSONLWriter struct {
	mu   sync.Mutex
	file *os.File
	path string
}

// NewJSONLWriter initializes or appends to a JSONL export file.
func NewJSONLWriter(path string) (*JSONLWriter, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed creating directory for %s: %w", path, err)
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed opening jsonl file %s: %w", path, err)
	}

	return &JSONLWriter{
		file: f,
		path: path,
	}, nil
}

// Write appends a record as a JSON line and flushes to disk.
func (w *JSONLWriter) Write(record interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed marshaling record to json: %w", err)
	}

	data = append(data, '\n')
	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("failed writing to jsonl file: %w", err)
	}

	return w.file.Sync()
}

// Close closes the underlying file handle.
func (w *JSONLWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
