package statuswriter

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

func WriteError(in *errors.StatusError, w http.ResponseWriter) {
	write(in.Status(), w)
}

func WriteStatus(in *metav1.Status, w http.ResponseWriter) {
	write(*in, w)
}

func write(in metav1.Status, w http.ResponseWriter) {
	outBytes, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader((int)(in.Code))
	w.Write(outBytes)
}
