package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"school_platform/db"
	"school_platform/routes"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db.Connect()

	mux := routes.RegisterRoutes()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Server starting on port", port)
	fmt.Println("Open http://localhost:8080 in your browser")

	err = http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
