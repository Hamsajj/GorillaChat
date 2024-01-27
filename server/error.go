package server

type ErrorResponse struct {
	Message string `json:"error"`
	Code    int    `json:"code"`
}

var ClientIDRequiredError = ErrorResponse{
	Message: "clientID is required",
	Code:    400,
}

var UnsupportedMessageType = ErrorResponse{
	Message: "message type is not supported",
	Code:    400,
}
