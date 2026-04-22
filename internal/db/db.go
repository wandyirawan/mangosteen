package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

type DB struct {
	*sql.DB
}

type Queries struct {
	db *DB
}

func Open(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func MustOpen(dsn string) *DB {
	db, err := Open(dsn)
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(dsn); os.IsNotExist(err) {
		schema, err := os.ReadFile("sql/schema.sql")
		if err != nil {
			panic(err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			panic(err)
		}
	}
	return db
}

func (db *DB) Query() *Queries {
	return &Queries{db}
}

func (db *DB) Close() error {
	return db.DB.Close()
}