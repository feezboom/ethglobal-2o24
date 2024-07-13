package main

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Message string `json:"message"`
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	response := Response{Message: "Hello, World!"}
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/", helloWorld)
	http.ListenAndServe(":8080", nil)
}
