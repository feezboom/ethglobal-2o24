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
	"time"
)

func generateRequestID() string {
	return uuid.New().String()
}

func submitQuestion(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	log.Printf("[%s] submitQuestion called", requestID)
	var req SubmitQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] Error decoding request: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Sender == "" || req.Question == "" || req.Receiver == "" || req.Signature == "" {
		log.Printf("[%s] Missing required fields in request", requestID)
		http.Error(w, "All fields (address, question, signature, receiver) are required", http.StatusBadRequest)
		return
	}

	checkSignature(req)

	nft, err := mintNft(req)
	if err != nil {
		log.Printf("[%s] Error minting question NFT: %v", requestID, err)
		http.Error(w, "error minting question nft", http.StatusBadRequest)
		return
	}

	var id string
	if req.Id != "" {
		id = req.Id
	} else {
		id = uuid.New().String()
	}

	q := Question{
		ID:              id,
		Question:        req.Question,
		Receiver:        strings.ToLower(req.Receiver),
		Sender:          strings.ToLower(req.Sender),
		Answered:        false,
		Signature:       req.Signature,
		TokenId:         nft.TokenID,
		CreatedAt:       time.Now().Format("2006-01-02 15:04:05.000"),
		ContractAddress: getContractAddress(),
	}

	_, err = questionsCollection.InsertOne(context.TODO(), q)
	if err != nil {
		log.Printf("[%s] Error inserting question into database: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[%s] Question submitted successfully: %+v", requestID, q)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

func checkSignature(_ SubmitQuestionRequest) {
	println("signature check done")
}

func listQuestionsForMe(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	log.Printf("[%s] listQuestionsForMe called", requestID)
	address := strings.ToLower(r.URL.Query().Get("address"))
	if address == "" {
		log.Printf("[%s] Missing 'address' query parameter", requestID)
		http.Error(w, "Sender query parameter is required", http.StatusBadRequest)
		return
	}

	cursor, err := questionsCollection.Find(context.TODO(), bson.M{"receiver": address, "contractAddress": getContractAddress()})
	if err != nil {
		log.Printf("[%s] Error finding questions for receiver: %v", requestID, err)
		if !errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			json.NewEncoder(w).Encode(new([]Question))
		}
		return
	}
	defer cursor.Close(context.TODO())

	questions := make([]Question, 0)
	if err := cursor.All(context.TODO(), &questions); err != nil {
		log.Printf("[%s] Error decoding questions: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[%s] Questions retrieved successfully for receiver: %s", requestID, address)

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}

func questionByID(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	log.Printf("[%s] questionByID called", requestID)
	id := strings.ToLower(r.URL.Query().Get("id"))

	if id == "" {
		log.Printf("[%s] Missing 'id' query parameter", requestID)
		http.Error(w, "Question ID is required", http.StatusBadRequest)
		return
	}

	q, err := findQuestionByIdAndContractAddress(id, getContractAddress())
	if err != nil {
		log.Printf("[%s] Error finding question by ID: %v", requestID, err)
		if !errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	log.Printf("[%s] Question retrieved successfully by ID: %s", requestID, id)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

func findQuestionByIdAndContractAddress(id, contractAddress string) (Question, error) {
	var q Question
	err := questionsCollection.FindOne(context.TODO(), bson.M{"id": id, "contractAddress": contractAddress}).Decode(&q)
	return q, err
}

func listQuestionsFromMe(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	log.Printf("[%s] listQuestionsFromMe called", requestID)
	address := strings.ToLower(r.URL.Query().Get("sender"))
	signature := r.URL.Query().Get("signature")
	if address == "" || signature == "" {
		log.Printf("[%s] Missing 'address' or 'signature' query parameters", requestID)
		http.Error(w, "Sender and signature query parameters are required", http.StatusBadRequest)
		return
	}

	cursor, err := questionsCollection.Find(context.TODO(), bson.M{"sender": address, "contractAddress": getContractAddress()})

	if err != nil {
		log.Printf("[%s] Error finding questions from sender: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	questions := make([]Question, 0)
	if err := cursor.All(context.TODO(), &questions); err != nil {
		log.Printf("[%s] Error decoding questions: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[%s] Questions retrieved successfully for sender: %s", requestID, address)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}

func submitAnswer(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	log.Printf("[%s] submitAnswer called", requestID)
	var req AnswerQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] Error decoding request: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.QuestionID == "" || req.Signature == "" || req.Answer == "" {
		log.Printf("[%s] Missing required fields in request", requestID)
		http.Error(w, "All fields (questionId, signature, answer) are required", http.StatusBadRequest)
		return
	}

	filter := bson.M{"id": req.QuestionID, "contractAddress": getContractAddress()}
	update := bson.M{
		"$set": bson.M{
			"answered": true,
			"answer":   req.Answer,
		},
	}
	_, err := questionsCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Printf("[%s] Error updating question with answer: %v", requestID, err)
		if !errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	q, err := findQuestionByIdAndContractAddress(req.QuestionID, getContractAddress())
	if err != nil {
		log.Printf("[%s] Error finding question by ID=%s after update: %v", requestID, req.QuestionID, err)
		if !errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	log.Printf("[%s] Question answered successfully: %+v", requestID, q)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
	w.WriteHeader(http.StatusOK)
}

func nftMetadata(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	log.Printf("[%s] nftMetadata called", requestID)
	vars := mux.Vars(r)
	tokenID := vars["tokenID"]

	if tokenID == "" {
		log.Printf("[%s] Missing 'tokenID' path parameter", requestID)
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

	err := questionsCollection.FindOne(context.TODO(), bson.M{
		"tokenID":         tokenID,
		"contractAddress": getContractAddress(),
	}).Decode(&q)

	if err != nil {
		log.Printf("[%s] Error finding question by tokenID=%s: %v", requestID, tokenID, err)
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
				Value:     strconv.FormatBool(q.Answered),
			},
		},
	}

	log.Printf("[%s] NFT metadata retrieved successfully for tokenID: %s", requestID, tokenID)
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
		description = "Q: ###encrypted###"
	}

	description += "\n\n\nSee your anonymous questions and answers here <a href=\"https://ask-fm-onchain-hackathon-jul-2024.fly.dev/\">MegaAsk: Ask.fm onchain</a>"

	return description
}
