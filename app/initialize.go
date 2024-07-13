package app

import "net/http"

func InitializeDbAndHandlers() {
	connectDB()

	http.HandleFunc("/api/submit-question", submitQuestion)
	http.HandleFunc("/api/questions", listQuestions)
	http.HandleFunc("/api/asked-questions", listAskedQuestions)
	http.HandleFunc("/api/answer-question", answerQuestion)
}
