package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Question struct {
	ID        string `json:"id,omitempty" bson:"id,omitempty"`
	Question  string `json:"question,omitempty" bson:"question,omitempty"`
	Receiver  string `json:"receiver,omitempty" bson:"receiver,omitempty"`
	Sender    string `json:"sender,omitempty" bson:"sender,omitempty"`
	Answered  bool   `json:"answered,omitempty" bson:"answered,omitempty"`
	Answer    string `json:"answer,omitempty" bson:"answer,omitempty"`
	Signature string `json:"signature,omitempty" bson:"signature,omitempty"`
}

type SubmitQuestionRequest struct {
	Address   string `json:"address"`
	Question  string `json:"question"`
	Signature string `json:"signature"`
	Receiver  string `json:"receiver"`
}

type AnswerQuestionRequest struct {
	QuestionID string `json:"questionId"`
	Signature  string `json:"signature"`
	Answer     string `json:"answer"`
}

var client *mongo.Client
var collection *mongo.Collection

func connectDB() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable is required")
	}

	clientOptions := options.Client().ApplyURI(mongoURI)

	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")
	collection = client.Database("testdb").Collection("questions")
}

func submitQuestion(w http.ResponseWriter, r *http.Request) {
	var req SubmitQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Address == "" || req.Question == "" || req.Receiver == "" || req.Signature == "" {
		http.Error(w, "All fields (address, question, signature, receiver) are required", http.StatusBadRequest)
		return
	}

	q := Question{
		ID:        fmt.Sprintf("%d", time.Now().Unix()),
		Question:  req.Question,
		Receiver:  req.Receiver,
		Sender:    req.Address,
		Answered:  false,
		Signature: req.Signature,
	}

	_, err := collection.InsertOne(context.TODO(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(q)
}

func listQuestions(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Address query parameter is required", http.StatusBadRequest)
		return
	}

	cursor, err := collection.Find(context.TODO(), bson.M{"receiver": address})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var questions []Question
	if err := cursor.All(context.TODO(), &questions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(questions)
}

func listAskedQuestions(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	signature := r.URL.Query().Get("signature")
	if address == "" || signature == "" {
		http.Error(w, "Address and signature query parameters are required", http.StatusBadRequest)
		return
	}

	cursor, err := collection.Find(context.TODO(), bson.M{"sender": address})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var questions []Question
	if err := cursor.All(context.TODO(), &questions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(questions)
}

func answerQuestion(w http.ResponseWriter, r *http.Request) {
	var req AnswerQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.QuestionID == "" || req.Signature == "" || req.Answer == "" {
		http.Error(w, "All fields (questionId, signature, answer) are required", http.StatusBadRequest)
		return
	}

	filter := bson.M{"id": req.QuestionID}
	update := bson.M{
		"$set": bson.M{
			"answered": true,
			"answer":   req.Answer,
		},
	}
	_, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	connectDB()

	http.HandleFunc("/api/submit-question", submitQuestion)
	http.HandleFunc("/api/questions", listQuestions)
	http.HandleFunc("/api/asked-questions", listAskedQuestions)
	http.HandleFunc("/api/answer-question", answerQuestion)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
