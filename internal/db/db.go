package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/projdocs/api/config"
)

var (
	instance *sql.DB
	once     sync.Once
	initErr  error
)

func Init() (*sql.DB, error) {
	once.Do(func() {
		db, err := sql.Open("pgx", config.MustGet().DatabaseURL)
		if err != nil {
			initErr = fmt.Errorf("opening db pool: %w", err)
			return
		}

		if err = db.Ping(); err != nil {
			db.Close()
			initErr = fmt.Errorf("pinging db: %w", err)
			return
		}

		var version string
		if err := db.QueryRow("SELECT version()").Scan(&version); err != nil {
			db.Close()
			initErr = fmt.Errorf("querying db version: %w", err)
			return
		}
		log.Printf("Database: %s", version)

		instance = db
	})

	return instance, initErr
}

func Get() (*sql.DB, error) {
	if instance == nil {
		return nil, errors.New("db is nil (was `db.Init` called?)")
	}
	return instance, nil
}

func MustGet() *sql.DB {
	if db, err := Get(); err != nil {
		panic(fmt.Sprintf("get db: %s", err.Error()))
	} else {
		return db
	}
}
