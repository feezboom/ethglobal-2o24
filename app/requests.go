package app

type SubmitQuestionRequest struct {
	Address   string `json:"address"`
	Question  string `json:"question"`
	Signature string `json:"signature"`
	Receiver  string `json:"receiver"`
}

type AnswerQuestionRequest struct {
	QuestionID string `json:"questionId"`
	Signature  string `json:"signature"`
	Answer     string `json:"answer"`
}
