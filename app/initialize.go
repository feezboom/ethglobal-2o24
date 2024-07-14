package app

import (
	"github.com/gorilla/mux"
	"net/http"
)

func InitializeDbAndHandlers() {
	connectDB()

	r := mux.NewRouter()
	r.HandleFunc("/api/submit-question", submitQuestion).Methods("POST")
	r.HandleFunc("/api/questions", listQuestionsForMe).Methods("GET")
	r.HandleFunc("/api/asked-questions", listQuestionsFromMe).Methods("GET")
	r.HandleFunc("/api/answer-question", submitAnswer).Methods("POST")
	r.HandleFunc("/api/question", questionByID).Methods("GET")

	r.HandleFunc("/{tokenID}", nftMetadata).Methods("GET")
	r.HandleFunc("/nft-metadata/{tokenID}", nftMetadata).Methods("GET")

	http.Handle("/", r)
}
