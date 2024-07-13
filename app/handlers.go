package app

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.mongodb.org/mongo-driver/bson"
	"math/big"
	"net/http"
	"os"
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

	nft := mintNft(req)

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

type NFT struct {
	ID       string `json:"id,omitempty" bson:"id,omitempty"`
	Contract string `json:"contract,omitempty" bson:"contract,omitempty"`
}

func mintNft(req SubmitQuestionRequest) (NFT, error) {
	privateKeyHex := os.Getenv("TECHNICAL_WALLET_PRIVATE_KEY")
	contractAddressHex := os.Getenv("NFT_CONTRACT_ADDRESS")
	rpcURL := os.Getenv("BLOCKCHAIN_RPC_URL")

	if privateKeyHex == "" || contractAddressHex == "" || rpcURL == "" {
		return NFT{}, fmt.Errorf("required environment variables are not set")
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return NFT{}, fmt.Errorf("invalid private key: %v", err)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return NFT{}, fmt.Errorf("failed to connect to the Ethereum mongoClient: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return NFT{}, fmt.Errorf("invalid public key")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return NFT{}, fmt.Errorf("failed to get nonce: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return NFT{}, fmt.Errorf("failed to get gas price: %v", err)
	}

	toAddress := common.HexToAddress(req.Receiver)
	tokenAddress := common.HexToAddress(contractAddressHex)

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1)) // Adjust the chain ID as needed
	if err != nil {
		return NFT{}, fmt.Errorf("failed to create authorized transactor: %v", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(3000000) // in units
	auth.GasPrice = gasPrice

	// Assuming the mint function is of the form mint(to address, tokenID uint256)
	tx, err := tokenAddress.Mint(auth, toAddress, big.NewInt(time.Now().Unix()))
	if err != nil {
		return NFT{}, fmt.Errorf("failed to mint NFT: %v", err)
	}

	nft := NFT{
		ID:       tx.Hash().Hex(),
		Contract: contractAddressHex,
	}

	return nft, nil
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
