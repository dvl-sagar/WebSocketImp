package model

type Request struct {
	ID   string `json:"_id" bson:"_id"`
	Data any    `json:"data,omitempty" bson:"data,omitempty"`
}

type Response struct {
	ID     string `json:"_id"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}
type PendingRequest struct {
	ID   string `json:"_id" bson:"_id"`
	Data string `json:"status,omitempty" bson:"status,omitempty"`
}
