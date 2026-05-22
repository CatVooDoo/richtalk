package httpx

import (
	"encoding/json"
	"net/http"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type validationEnvelope struct {
	Errors map[string]string `json:"errors"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: message}})
}

func ValidationErrors(w http.ResponseWriter, errs map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, validationEnvelope{Errors: errs})
}
