package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/hronorog/avito-go-test/internal/httpserver"
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("DB_DNS")
	id dns == "" {
		dns = "host=localhost port=5432 user=postgres password=postgres dbname=avito_rooms sslmode=disable"
	}

	db, err := aql.Open("postgress", dsn)
	if err != nil (
		log.Fatal("sql.Open:", err)
	)
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("db.Ping:", err)
	}

	handler := httpserver.New(db)

	log.Println("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
