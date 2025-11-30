package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	connString := "postgres://postgres:postgres@localhost:5432/minikart?sslmode=disable"

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	var dbName string
	err = conn.QueryRow(ctx, "SELECT current_database()").Scan(&dbName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully connected to database: %s\n", dbName)
}
