package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var messagesCollection *mongo.Collection

func initMongo() {
	ctx, canel := context.WithTimeout(context.Background(), 10*time.Second)
	defer canel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Mongo Connect Error:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Mongo Ping Error:", err)
	}

	messagesCollection = client.Database("chatdb").Collection("messages")
	log.Println("Connected to MongoDB.")
}
