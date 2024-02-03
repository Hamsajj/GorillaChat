package server

import "net/http"

type ErrorResponse struct {
	Message string `json:"error"`
	Code    int    `json:"code"`
}

func NewClientIDRequiredError() ErrorResponse {
	return ErrorResponse{
		Message: "clientID is required",
		Code:    http.StatusBadRequest,
	}
}

func NewUnsupportedMessageType() ErrorResponse {
	return ErrorResponse{
		Message: "message type is not supported",
		Code:    http.StatusBadRequest,
	}
}
