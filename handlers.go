package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

func handleChat(w http.ResponseWriter, r *http.Request) {
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
	}
	log.WithFields(log.Fields{
		"method": r.Method,
		"path":   r.RequestURI,
		"ip":     r.RemoteAddr,
		"body":   string(bodyBytes),
	}).Info("Incoming request")
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("Error decoding request: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	cur, err := messagesCollection.Find(ctx, bson.M{"chat_id": req.ChatID})
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	var history []Message
	for cur.Next(ctx) {
		var msg Message
		if err := cur.Decode(&msg); err == nil {
			history = append(history, msg)
		}
	}

	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp.Before(history[j].Timestamp)
	})

	imageToUse := req.Image
	if imageToUse == "" {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i].Image != "" {
				imageToUse = history[i].Image
				log.Info("Fallback to image from history")
				break
			}
		}
	}

	userMsg := Message{
		ChatID:    req.ChatID,
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
		UserID:    req.UserID,
		Image:     imageToUse,
	}
	history = append(history, userMsg)

	objectValue := req.Object
	if objectValue == "" {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i].Role == "user" && history[i].UserID != "" {
				objectValue = history[i].UserID
				break
			}
		}
	}
	log.WithField("Object", objectValue).Info("Resolved object value")

	var promptBuilder strings.Builder
	promptBuilder.WriteString("Chat history:\n")
	for _, msg := range history {
		role := strings.Title(msg.Role)
		promptBuilder.WriteString(role + ": " + msg.Content + "\n")
	}
	promptBuilder.WriteString("\nPrompt: " + req.Message)
	finalPrompt := promptBuilder.String()

	apiPayload := map[string]interface{}{
		"prompt": finalPrompt,
		"object": objectValue,
	}
	if imageToUse != "" {
		apiPayload["image"] = imageToUse
		log.Info("Including image in LLM payload")
	}

	payloadBytes, _ := json.Marshal(apiPayload)

	log.WithField("payload", string(payloadBytes)).Info("Sending request to external LLM")

	resp, err := http.Post("https://a91f-77-76-10-114.ngrok-free.app/chat", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Baldr API error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var baldrResponse struct {
		Content string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&baldrResponse); err != nil {
		log.Errorf("Error decoding response from LLM: %v", err)
		http.Error(w, "LLM response error", http.StatusInternalServerError)
		return
	}

	assistantMsg := Message{
		ChatID:    req.ChatID,
		Role:      "assistant",
		Content:   baldrResponse.Content,
		Timestamp: time.Now(),
		UserID:    req.UserID,
	}

	json.NewEncoder(w).Encode(assistantMsg)

	if req.UserID != "" {
		_, err = messagesCollection.InsertMany(ctx, []interface{}{userMsg, assistantMsg})
		if err != nil {
			log.Errorf("DB insert error: %v", err)
			http.Error(w, "DB insert error", http.StatusInternalServerError)
			return
		}
	}
}

func getChatHistory(w http.ResponseWriter, r *http.Request) {
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

func handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	log.Info("Received new /auth request")

	var req GoogleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("Failed to decode auth request body")
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	log.WithField("code", req.Code).Info("Parsed authorization code from frontend")

	data := map[string]string{
		"code":          req.Code,
		"client_id":     os.Getenv("GOOGLE_CLIENT_ID"),
		"client_secret": os.Getenv("GOOGLE_CLIENT_SECRET"),
		"redirect_uri":  os.Getenv("GOOGLE_REDIRECT_URI"),
		"grant_type":    "authorization_code",
	}
	payloadBytes, _ := json.Marshal(data)

	log.WithFields(log.Fields{
		"code":         req.Code,
		"client_id":    os.Getenv("GOOGLE_CLIENT_ID"),
		"redirect_uri": os.Getenv("GOOGLE_REDIRECT_URI"),
		"grant_type":   "authorization_code",
	}).Info("Sending request to Google OAuth2")

	reqGoogle, _ := http.NewRequest("POST", "https://oauth2.googleapis.com/token", bytes.NewBuffer(payloadBytes))
	reqGoogle.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(reqGoogle)
	if err != nil {
		log.WithError(err).Error("Failed to send request to Google OAuth server")
		http.Error(w, "Failed to contact Google", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.WithField("status", resp.StatusCode).Info("Received response from Google OAuth server")

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.WithField("google_error_response", string(body)).Error("Google OAuth2 error response")
		http.Error(w, "Failed to exchange code", http.StatusBadRequest)
		return
	}

	var googleResp GoogleTokenResponse
	if err := json.Unmarshal(body, &googleResp); err != nil {
		log.WithError(err).Error("Failed to parse Google's token response")
		http.Error(w, "Invalid response from Google", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"access_token_present": googleResp.AccessToken != "",
		"id_token_present":     googleResp.IDToken != "",
	}).Info("Successfully exchanged code for tokens")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(googleResp)
}

func getUserMessages(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimPrefix(r.URL.Path, "/user/")
	userID = strings.TrimSuffix(userID, "/")
	if userID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}
	cursor, err := messagesCollection.Find(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to find messages")
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var messages []Message
	if err := cursor.All(ctx, &messages); err != nil {
		log.WithError(err).Error("Failed to decode messages")
		http.Error(w, "Failed to parse messages", http.StatusInternalServerError)
		return
	}

	log.WithField("user_id", userID).Infof("Fetched %d messages", len(messages))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
