package main

import (
	"log"
	"net/http"
	"os"

	"github.com/hronorog/avito-go-test/internal/httpserver"
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	handler := httpserver.New()

	log.Println("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
