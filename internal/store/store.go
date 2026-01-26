package store

import "fmt"

// DB is a minimal database handle used by the controller in tests
type DB struct {
	Path string
}

// NewDB creates a new DB handle (stub implementation)
func NewDB(path string) (*DB, error) {
	if path == "" {
		return nil, fmt.Errorf("empty database path")
	}
	return &DB{Path: path}, nil
}

// Close closes the database (no-op for stub)
func (d *DB) Close() error {
	return nil
}
