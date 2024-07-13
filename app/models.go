package app

type Question struct {
	ID        string `json:"id,omitempty" bson:"id,omitempty"`
	Question  string `json:"question,omitempty" bson:"question,omitempty"`
	Receiver  string `json:"receiver,omitempty" bson:"receiver,omitempty"`
	Sender    string `json:"sender,omitempty" bson:"sender,omitempty"`
	Answered  bool   `json:"answered,omitempty" bson:"answered,omitempty"`
	Answer    string `json:"answer,omitempty" bson:"answer,omitempty"`
	Signature string `json:"signature,omitempty" bson:"signature,omitempty"`
	TokenID   string `json:"tokenID,omitempty" bson:"tokenID,omitempty"`
}
