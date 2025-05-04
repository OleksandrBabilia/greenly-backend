package main

import "time"

type Message struct {
	ChatID    string    `json:"chat_id" bson:"chat_id"`
	Role      string    `json:"role" bson:"role"`
	Content   string    `json:"content" bson:"content"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	UserID    string    `json:"user_id" bson:"user_id"`
	Object    string    `json:"object_name"`
	Image     string    `bson:"image,omitempty" json:"image,omitempty"`
	ImageName string    `bson:"image_name,omitempty" json:"image_name,omitempty"`
}

type ChatRequest struct {
	ChatID      string    `json:"chat_id"`
	Message     string    `json:"content"`
	UserID      string    `json:"user_id"`
	Object      string    `json:"object_name"`
	ChatHistory []Message `json:"chat_history,omitempty"`
	Image       string    `json:"image,omitempty"`
	ImageName   string    `json:"image_name,omitempty"`
}

type GoogleAuthRequest struct {
	Code string `json:"code"`
}

type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
}
