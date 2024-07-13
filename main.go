package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Response struct {
	Message string `json:"message"`
}

type Entry struct {
	ID      string `json:"id,omitempty" bson:"id,omitempty"`
	Content string `json:"content,omitempty" bson:"content,omitempty"`
}

var client *mongo.Client
var collection *mongo.Collection

func connectDB() {
	var err error
	clientOptions := options.Client().ApplyURI("your-amazon-documentdb-uri")

	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")
	collection = client.Database("testdb").Collection("entries")
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	response := Response{Message: "Hello, World 1!"}
	json.NewEncoder(w).Encode(response)
}

func addEntry(w http.ResponseWriter, r *http.Request) {
	var entry Entry
	_ = json.NewDecoder(r.Body).Decode(&entry)

	entry.ID = fmt.Sprintf("%d", time.Now().Unix())
	_, err := collection.InsertOne(context.TODO(), entry)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(entry)
}

func main() {
	connectDB()

	http.HandleFunc("/", helloWorld)
	http.HandleFunc("/add", addEntry)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
