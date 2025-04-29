package main

import (
	"net/http"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

func main() {
	godotenv.Load()
	initMongo()
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)

	mux := http.NewServeMux()
	mux.HandleFunc("/chat", handleChat)
	mux.HandleFunc("/chat/", getChatHistory)
	mux.HandleFunc("/user/", getUserMessages)
	mux.HandleFunc("/auth", handleGoogleAuth)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://192.168.0.102:3000", "http://192.168.0.102.nip.io:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true, // emm
	})

	handler := loggingMiddleware(c.Handler(mux))

	log.Println("Server started on :8081")
	log.Fatal(http.ListenAndServe(":8081", handler))
}
