package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ParquetWriter batches records for columnar analytics ingestion.
type ParquetWriter struct {
	mu   sync.Mutex
	path string
	file *os.File
}

// NewParquetWriter initializes a columnar destination writer.
func NewParquetWriter(path string) (*ParquetWriter, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed creating directory for %s: %w", path, err)
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed opening parquet file %s: %w", path, err)
	}

	return &ParquetWriter{
		path: path,
		file: f,
	}, nil
}

// Write appends an entity record.
func (p *ParquetWriter) Write(record interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = p.file.Write(data)
	return err
}

// Close closes the file handle.
func (p *ParquetWriter) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.file != nil {
		return p.file.Close()
	}
	return nil
}
