package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClient *mongo.Client
var questionsCollection *mongo.Collection
var nftIdCollection *mongo.Collection

func connectDB() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable is required")
	}

	mongoClient, err = mongo.NewClient(options.Client().ApplyURI(mongoURI).SetTLSConfig(&tls.Config{
		InsecureSkipVerify: true, // DocumentDB requires this setting for TLS connections
	}))
	if err != nil {
		log.Fatalf("Failed to create mongoClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}

	fmt.Println("Connected to DocumentDB!")

	questionsCollection = mongoClient.Database("testdb").Collection("questions")
	nftIdCollection = mongoClient.Database("testdb").Collection("nftIdCollection")
}
