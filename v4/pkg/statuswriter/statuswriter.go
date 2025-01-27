package statuswriter

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"net/http"
)

func WriteError(in *errors.StatusError, w http.ResponseWriter) {
	write(in.Status(), w)
}

func WriteStatus(in *metav1.Status, w http.ResponseWriter) {
	write(*in, w)
}

func WriteSuccess(msg string, w http.ResponseWriter) {
	write(metav1.Status{
		Status:  metav1.StatusSuccess,
		Message: msg,
		Details: nil,
		Code:    http.StatusOK,
	}, w)
}

func write(in metav1.Status, w http.ResponseWriter) {
	outBytes, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader((int)(in.Code))
	if _, err := w.Write(outBytes); err != nil {
		slog.Error("error writing http response", "error", err.Error())
	}
}
