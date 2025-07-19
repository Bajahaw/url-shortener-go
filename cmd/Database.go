package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"os"
	"strings"
)

const (
	DatabaseUrl = "postgres://user:pass@localhost:5432/urls?sslmode=disable"
)

func ConnectDB() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, DatabaseUrl)
	if err != nil {
		log.Println("DB Connection Failed:", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)
	log.Println("DB Connected Successfully")
	createTable(ctx, conn)
}

func createTable(ctx context.Context, conn *pgx.Conn) {
	sql := "CREATE TABLE urls (id VARCHAR(10) PRIMARY KEY, url TEXT NOT NULL);"
	_, err := conn.Exec(ctx, sql)
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
