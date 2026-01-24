package model

type BBErrorResponse struct {
	Type             string `json:"type"`
	Title            string `json:"title"`
	Status           int    `json:"status"`
	Detail           string `json:"detail"`
	CorrelationID    string `json:"correlationId"`
	NumeroOcorrencia int    `json:"numeroOcorrencia"`
}
