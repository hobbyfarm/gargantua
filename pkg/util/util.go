package util

import (
	"net/http"
	"strconv"
	"encoding/json"
)

func ReturnHTTPMessage(w http.ResponseWriter, r *http.Request, httpStatus int, messageType string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	err := HTTPMessage{
		Status:  strconv.Itoa(httpStatus),
		Message: message,
		Type:    messageType,
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(err)
}

func GetHTTPErrorCode(httpStatus int) string {
	switch httpStatus {
	case 401:
		return "Unauthorized"
	case 404:
		return "NotFound"
	case 403:
		return "PermissionDenied"
	case 500:
		return "ServerError"
	}

	return "ServerError"
}

type HTTPMessage struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}