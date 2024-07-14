package app

type Question struct {
	ID              string `json:"id,omitempty" bson:"id,omitempty"`
	Question        string `json:"question,omitempty" bson:"question,omitempty"`
	Receiver        string `json:"receiver,omitempty" bson:"receiver,omitempty"`
	Sender          string `json:"sender,omitempty" bson:"sender,omitempty"`
	Answered        bool   `json:"answered,omitempty" bson:"answered,omitempty"`
	Answer          string `json:"answer,omitempty" bson:"answer,omitempty"`
	Signature       string `json:"signature,omitempty" bson:"signature,omitempty"`
	TokenId         string `json:"tokenId,omitempty" bson:"tokenID,omitempty"`
	CreatedAt       string `json:"createdAt,omitempty" bson:"createdAt,omitempty"`
	ContractAddress string `json:"contractAddress,omitempty" bson:"contractAddress,omitempty"`
}

type SubmitQuestionRequest struct {
	Id        string `json:"id"`
	Sender    string `json:"sender"`
	Question  string `json:"question"`
	Signature string `json:"signature"`
	Receiver  string `json:"receiver"`
}

type AnswerQuestionRequest struct {
	QuestionID string `json:"questionId"`
	Signature  string `json:"signature"`
	Answer     string `json:"answer"`
}
