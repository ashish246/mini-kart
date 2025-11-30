package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	// Try connecting to postgres database first
	connString := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to postgres database: %v\n", err)
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

	// List all databases
	rows, err := conn.Query(ctx, "SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	fmt.Println("\nAvailable databases:")
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  - %s\n", name)
	}
}
