package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

func main() {
	// Context
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	appSignal := make(chan os.Signal, 3)
	signal.Notify(appSignal, os.Interrupt)

	go func() {
		<-appSignal
		stop()
	}()

	dsn := "sqlserver://SA:myStrong(!)Password@localhost:1433?database=tempdb"

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(3)
	db.SetMaxOpenConns(3)

	// Open connection
	OpenDbConnection(ctx, db)

	// Drop table if exists
	_, err = db.Exec("DROP TABLE IF EXISTS employee")
	if err != nil {
		log.Fatal(err)
	}

	// Create table
	_, err = db.Exec(`
		CREATE TABLE employee (
		    id INT,
			name VARCHAR(20),
			start_dt DATETIME,
			is_remote BIT
		)`)
	if err != nil {
		log.Fatal(err)
	}

	// Insert some data in table
	InsertData(ctx, db)

	// Error handling and transactions
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO employee (id, name, start_dt, is_remote)
		VALUES
		    (3000000000, 'id int64 instead of int32', '2022-06-17 11:00:00', 1)`)
	if err != nil {
		log.Printf("ERROR: %s\n", err.Error()) // Do not fail, just print the error in output
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	// Query single row
	var version string
	if err = db.QueryRow("SELECT @@version").Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			log.Println("No rows found.")
		} else {
			log.Fatalf("unable to execute query: %v", err)
		}
	} else {
		fmt.Printf("SERVER VERSION: %s\n", version)
	}

	// Query multiple rows
	QueryAndPrintData(ctx, db)
}

func OpenDbConnection(ctx context.Context, db *sql.DB) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal("Unable to connect to database: %v", err)
	}
}

func InsertData(ctx context.Context, db *sql.DB) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := db.ExecContext(timeout, `
		INSERT INTO employee (id, name, start_dt, is_remote) 
		VALUES
		    (1, 'John Doe', '2022-01-01 09:00:00', 1),
       		(2, 'Jane Smith', '2023-03-15 10:00:00', 0)`)
	if err != nil {
		log.Fatal(err)
	}
}

func QueryAndPrintData(ctx context.Context, db *sql.DB) {
	var id int
	var name string
	var startDt time.Time
	var isRemote bool

	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(timeout, "SELECT id, name, start_dt, is_remote FROM employee")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Print the results
	fmt.Println("Results:")
	for rows.Next() {
		err = rows.Scan(&id, &name, &startDt, &isRemote)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("ID:", id, "Name:", name, "Start Datetime:", startDt, "Is Remote:", isRemote)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}
