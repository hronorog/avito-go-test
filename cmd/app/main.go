package main

import (
    "database/sql"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    _ "github.com/lib/pq"
    "github.com/hronorog/avito-go-test/internal/httpserver"
)


func main() {
    port := os.Getenv("APP_PORT")
    if port == "" {
        port = "8080"
    }

    // сначала пробуем DB_DSN целиком, если задан
    dsn := os.Getenv("DB_DSN")
    if dsn == "" {
        host := os.Getenv("DB_HOST")
        if host == "" {
            host = "localhost"
        }
        portDB := os.Getenv("DB_PORT")
        if portDB == "" {
            portDB = "5432"
        }
        user := os.Getenv("DB_USER")
        if user == "" {
            user = "postgres"
        }
        pass := os.Getenv("DB_PASSWORD")
        if pass == "" {
            pass = "123456"
        }
        name := os.Getenv("DB_NAME")
        if name == "" {
            name = "avito_rooms"
        }

        dsn = fmt.Sprintf(
            "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
            host, portDB, user, pass, name,
        )
    }

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("sql.Open:", err)
	}
	defer db.Close()

	var pingErr error
	for i := 0; i < 10; i++ {
		pingErr = db.Ping()
		if pingErr == nil {
			break
		}
		log.Printf("db.Ping attempt %d failed: %v", i+1, pingErr)
		time.Sleep(2 * time.Second)
	}
	if pingErr != nil {
		log.Fatal("db.Ping:", pingErr)
	}

    handler := httpserver.New(db)

    log.Printf("listening on :%s", port)
    if err := http.ListenAndServe(":"+port, handler); err != nil {
        log.Fatal(err)
    }
}
