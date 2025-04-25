package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

func handleChat(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{
		"method": r.Method,
		"path":   r.RequestURI,
		"ip":     r.RemoteAddr,
		"body":   r.Body,
	}).Info("incoming request")

	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}

	ctx := context.Background()

	cur, err := messagesCollection.Find(ctx, bson.M{"chat_id": req.ChatID})
	if err != nil {
		http.Error(w, "DB error", 500)
		return
	}
	defer cur.Close(ctx)

	var history []Message
	for cur.Next(ctx) {
		var msg Message
		cur.Decode(&msg)
		history = append(history, msg)
	}

	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp.Before(history[j].Timestamp)
	})

	userMsg := Message{
		ChatID:    req.ChatID,
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
	}
	history = append(history, userMsg)

	apiPayload := map[string]interface{}{
		"message": history,
	}
	payloadBytes, _ := json.Marshal(apiPayload)

	resp, err := http.Post("http://localhost:5000/chat", "application/json", bytes.NewBuffer(payloadBytes))

	if err != nil {
		http.Error(w, "Baldr API Error", 500)
		return
	}
	defer resp.Body.Close()

	var baldrResponse struct {
		Content string `json:"content"`
	}
	json.NewDecoder(resp.Body).Decode(&baldrResponse)

	assistantMsg := Message{
		ChatID:    req.ChatID,
		Role:      "assistant",
		Content:   baldrResponse.Content,
		Timestamp: time.Now(),
	}
	_, err = messagesCollection.InsertMany(ctx, []interface{}{userMsg, assistantMsg})
	if err != nil {
		http.Error(w, "DB insert error", 500)
		return
	}

	json.NewEncoder(w).Encode((assistantMsg))
}

func getChatHistort(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{
		"method": r.Method,
		"path":   r.RequestURI,
		"ip":     r.RemoteAddr,
		"body":   r.Body,
	}).Info("incoming request")

	chatID := strings.TrimPrefix(r.URL.Path, "/chat/")
	chatID = strings.TrimSuffix(chatID, "/")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := messagesCollection.Find(ctx, bson.M{"chat_id": chatID})
	if err != nil {
		http.Error(w, "Failed to retrive messages", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var messages []Message
	if err := cursor.All(ctx, &messages); err != nil {
		http.Error(w, "Failed to decode messages", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}
