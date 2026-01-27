package database

import (
	"database/sql"
)

func NewSQLiteDB(path string) (*sql.DB, error) {
	if path == "" {
		path = ":memory:"
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return db, nil
}
