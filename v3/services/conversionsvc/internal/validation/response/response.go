package response

import (
	"fmt"
	v12 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func RespDenied(msg string, fields ...interface{}) *v12.AdmissionResponse {
	return &v12.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: fmt.Sprintf(msg, fields...),
		},
	}
}
