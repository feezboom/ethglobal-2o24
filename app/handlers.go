package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"strings"
	"time"
)

func submitQuestion(w http.ResponseWriter, r *http.Request) {
	var req SubmitQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Sender == "" || req.Question == "" || req.Receiver == "" || req.Signature == "" {
		http.Error(w, "All fields (address, question, signature, receiver) are required", http.StatusBadRequest)
		return
	}

	checkSignature(req)

	nft, err := mintNft(req)
	if err != nil {
		http.Error(w, "error minting question nft", http.StatusBadRequest)
		log.Fatal(err)
		return
	}

	var id string
	if req.Id != "" {
		id = req.Id
	} else {
		id = fmt.Sprintf("%d", time.Now().Unix())
	}

	q := Question{
		ID:        id,
		Question:  req.Question,
		Receiver:  strings.ToLower(req.Receiver),
		Sender:    strings.ToLower(req.Sender),
		Answered:  false,
		Signature: req.Signature,
		TokenID:   nft.TokenID,
	}

	_, err = questionsCollection.InsertOne(context.TODO(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(q)
}

func checkSignature(_ SubmitQuestionRequest) {
	println("signature check done")
}

func listQuestionsForMe(w http.ResponseWriter, r *http.Request) {
	address := strings.ToLower(r.URL.Query().Get("address"))
	if address == "" {
		http.Error(w, "Sender query parameter is required", http.StatusBadRequest)
		return
	}

	cursor, err := questionsCollection.Find(context.TODO(), bson.M{"receiver": address})
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			json.NewEncoder(w).Encode(new([]Question))
			return
		}

	}
	defer cursor.Close(context.TODO())

	var questions []Question
	if err := cursor.All(context.TODO(), &questions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(questions)
}

func listQuestionsFromMe(w http.ResponseWriter, r *http.Request) {
	address := strings.ToLower(r.URL.Query().Get("address"))
	signature := r.URL.Query().Get("signature")
	if address == "" || signature == "" {
		http.Error(w, "Sender and signature query parameters are required", http.StatusBadRequest)
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

func nftMetadata(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tokenID := vars["tokenID"]

	if tokenID == "" {
		http.Error(w, "NFT token ID is required", http.StatusBadRequest)
		return
	}

	type NftAttribute struct {
		TraitType string `json:"trait_type"`
		Value     string `json:"value"`
	}

	type ResponseNft struct {
		Name        string         `json:"id"`
		Description string         `json:"description"`
		Image       string         `json:"image"`
		Attributes  []NftAttribute `json:"attributes"`
	}

	var q Question

	err := questionsCollection.FindOne(context.TODO(), bson.M{"tokenID": tokenID}).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatalf("error finding question: %v", err)
		return
	}

	nft := ResponseNft{
		Name:        "New Question",
		Description: q.Question,
		Image:       "https://files.slack.com/files-pri/T3V7DQ6HW-F07C9P67HLJ/nft_ask.png",
		Attributes:  nil,
	}

	json.NewEncoder(w).Encode(nft)
}
