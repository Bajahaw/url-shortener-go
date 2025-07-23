package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
)

var (
	ctx         context.Context
	pool        *pgxpool.Pool
	DatabaseUrl = os.Getenv("DATABASE_URL")
)

func ConnectDB() {
	ctx = context.Background()
	var err error
	pool, err = pgxpool.New(ctx, DatabaseUrl)
	if err != nil {
		log.Println("DB Connection Failed:", err)
		os.Exit(1)
	}
	log.Println("DB Connected Successfully")
	createTable()
}

func createTable() {
	sql := "CREATE TABLE urls (id VARCHAR(10) PRIMARY KEY, url TEXT NOT NULL)"
	_, err := pool.Exec(ctx, sql)
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

func GetURL(id string) (string, error) {
	sql := "SELECT url FROM urls WHERE id = $1"
	row := pool.QueryRow(ctx, sql, id)
	var url string
	err := row.Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

func SaveURL(id, url string) error {
	sql := "INSERT INTO urls (id, url) VALUES ($1, $2)"
	_, err := pool.Exec(ctx, sql, id, url)
	if err != nil {
		return err
	}
	return nil
}
