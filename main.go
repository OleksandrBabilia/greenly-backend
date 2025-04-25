package main

import (
	"net/http"

	"github.com/rs/cors"

	log "github.com/sirupsen/logrus"
)

func main() {
	initMongo()
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)

	mux := http.NewServeMux()
	mux.HandleFunc("/chat", handleChat)
	mux.HandleFunc("/chat/", getChatHistort)
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}).Handler(mux)

	log.Println("Server started on:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
