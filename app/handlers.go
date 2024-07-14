package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
		id = uuid.New().String()
	}

	q := Question{
		ID:        id,
		Question:  req.Question,
		Receiver:  strings.ToLower(req.Receiver),
		Sender:    strings.ToLower(req.Sender),
		Answered:  false,
		Signature: req.Signature,
		TokenId:   nft.TokenID,
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
		} else {
			json.NewEncoder(w).Encode(new([]Question))
		}
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

func questionByID(w http.ResponseWriter, r *http.Request) {
	id := strings.ToLower(r.URL.Query().Get("id"))

	if id == "" {
		http.Error(w, "Question ID is required", http.StatusBadRequest)
		return
	}

	q, err := findQuestionById(id)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

func findQuestionById(id string) (Question, error) {
	var q Question
	err := questionsCollection.FindOne(context.TODO(), bson.M{"id": id}).Decode(&q)
	return q, err
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

func submitAnswer(w http.ResponseWriter, r *http.Request) {
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
		if !errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	q, err := findQuestionById(req.QuestionID)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	json.NewEncoder(w).Encode(q)
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
		Value     any    `json:"value"`
	}

	type ResponseNft struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Image       string         `json:"image"`
		Attributes  []NftAttribute `json:"attributes"`
	}

	var q Question

	err := questionsCollection.FindOne(context.TODO(), bson.M{"tokenID": tokenID}).Decode(&q)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			json.NewEncoder(w).Encode(ResponseNft{})
		}

		return
	}

	nft := ResponseNft{
		Name:        "New Question",
		Description: buildDescription(q),
		Image:       buildImage(q),
		Attributes: []NftAttribute{
			{
				TraitType: "IsAnswered",
				Value:     strconv.FormatBool(true),
			},
		},
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nft)
}

func buildImage(q Question) string {
	if q.Answered {
		return "https://expression-statement.fly.dev/ask-nft?text=" + url.QueryEscape(q.Question)
	}
	return "ipfs://QmNSJtpv8W85T3ZSPtmaZvSS3bK8jp7Pus36qT8beEE42e"
}

func buildDescription(q Question) string {

	description := q.Question
	if q.Answered {
		description = "Q:" + description + "\nAnswer: " + q.Answer
	} else {
		description = "Q: ###encrypted###" + description
	}

	return description
}
