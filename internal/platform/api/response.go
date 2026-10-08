package api

import (
	"encoding/json/v2"
	"net/http"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func WriteValidationError(
	w http.ResponseWriter,
	details map[string]string,
) {
	WriteJSON(w, http.StatusBadRequest, ErrorResponse{
		Error: ErrorBody{
			Code:    "validation_error",
			Message: "request validation failed",
			Details: details,
		},
	})
}

func WriteError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
) {
	WriteJSON(w, status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func WriteJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.MarshalWrite(w, value)
}
