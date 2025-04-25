package main

import "time"

type Message struct {
	ChatID    string    `json:"chat_id" bson:"chat_id"`
	Role      string    `json:"role" bson:"role"`
	Content   string    `json:"contenct" bson:"content"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

type ChatRequest struct {
	ChatID  string `json:"chat_id"`
	Message string `json:"message"`
}
