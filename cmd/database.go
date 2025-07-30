package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"time"
)

var (
	ctx         context.Context
	pool        *pgxpool.Pool
	databaseUrl = os.Getenv("DATABASE_URL")
)

type Repository interface {
	SaveURL(id, url string) error
	GetURL(id string) (string, error)
}
type Database struct{}

func NewRepository() Repository {
	return &Database{}
}

func ConnectDB() {
	ctx = context.Background()
	var err error
	pool, err = pgxpool.New(ctx, databaseUrl)
	if err != nil {
		log.Println("DB Connection Failed:", err)
		os.Exit(1)
	}
	log.Println("DB Connected Successfully")
	createTable()
}

func createTable() {
	sql := "CREATE TABLE urls (id VARCHAR(10) PRIMARY KEY, url TEXT NOT NULL)"
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := pool.Exec(queryCtx, sql)
	if err != nil {
		if strings.Contains(err.Error(), "42P07") {
			log.Println("Table urls already exist!")
		} else {
			log.Println("Failed to create table:", err)
		}
	} else {
		log.Println("Table created")
	}
}

func (db *Database) GetURL(id string) (string, error) {
	sql := "SELECT url FROM urls WHERE id = $1"
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := pool.QueryRow(queryCtx, sql, id)
	var url string
	err := row.Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (db *Database) SaveURL(id, url string) error {
	sql := "INSERT INTO urls (id, url) VALUES ($1, $2)"
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := pool.Exec(queryCtx, sql, id, url)
	if err != nil {
		return err
	}
	return nil
}
