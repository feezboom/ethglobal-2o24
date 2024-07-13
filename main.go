package main

import (
	"ethglobal-2o24/app"
	"log"
	"net/http"
	"os"
)

func main() {
	app.InitializeDbAndHandlers()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
