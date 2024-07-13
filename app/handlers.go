package app

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"time"
)

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

	checkSignature(req)

	nft, err := mintNft(req)
	if err != nil {
		http.Error(w, "error minting question nft", http.StatusBadRequest)
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

	_, err := questionsCollection.InsertOne(context.TODO(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(q)
}

func checkSignature(_ SubmitQuestionRequest) {
	println("signature check")
}

func listQuestions(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Address query parameter is required", http.StatusBadRequest)
		return
	}

	cursor, err := questionsCollection.Find(context.TODO(), bson.M{"receiver": address})
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

	cursor, err := questionsCollection.Find(context.TODO(), bson.M{"sender": address})
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
	_, err := questionsCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
