package main

import (
	"context"
	"database/sql"
	"github.com/jackc/pgx/v5"
	"os"
)

var db sql.DB

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
}
