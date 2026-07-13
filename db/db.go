package db

import (
    "database/sql"
    "fmt"
    "log"
    "os"

    _ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {
    dbURL := os.Getenv("DB_URL")
    if dbURL == "" {
        log.Fatal("DB_URL is not set in .env")
    }

    var err error
    DB, err = sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatal("Failed to open database connection:", err)
    }

    err = DB.Ping()
    if err != nil {
        log.Fatal("Cannot reach the database:", err)
    }

    fmt.Println("Database connected successfully!")
}