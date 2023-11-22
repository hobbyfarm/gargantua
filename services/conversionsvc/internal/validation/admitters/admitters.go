package admitters

import (
	"context"
	v12 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
)

type V1Review func(context.Context, *v12.AdmissionRequest) *v12.AdmissionResponse

type V1beta1Review func(ctx context.Context, review *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse

type Admitters struct {
	V1      V1Review
	V1beta1 V1beta1Review
}
