package app

import "net/http"

func InitializeDbAndHandlers() {
	connectDB()

	http.HandleFunc("/api/submit-question", submitQuestion)
	http.HandleFunc("/api/questions", listQuestionsForMe)
	http.HandleFunc("/api/asked-questions", listQuestionsFromMe)
	http.HandleFunc("/api/answer-question", answerQuestion)
}
