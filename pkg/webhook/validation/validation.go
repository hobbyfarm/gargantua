package validation

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"github.com/hobbyfarm/gargantua/pkg/webhook/validation/admitters"
	"github.com/hobbyfarm/gargantua/pkg/webhook/validation/setting"
	"github.com/pkg/errors"
	"io"
	v12 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"net/http"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

var (
	handlers = map[schema.GroupVersionKind]admitters.Admitters{}
)

type register func() (schema.GroupVersionKind, admitters.Admitters)

func init() {
	runtimeScheme.AddKnownTypes(v12.SchemeGroupVersion,
		&v12.AdmissionReview{})

	for _, f := range []register{
		setting.Register,
	} {
		g, a := f()
		handlers[g] = a
	}
}

func RegisterRoutes(router *mux.Router) {
	for k, _ := range handlers {
		router.Path(fmt.Sprintf("/%s/%s/%s", k.Group, k.Version, k.Kind)).
			HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				dispatch(k, writer, request)
			})
	}
}

func dispatch(gvk schema.GroupVersionKind, w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		glog.Error(errors.Wrap(err, "error reading request body of validating review"))
		return
	}

	obj, requestGvk, err := deserializer.Decode(body, nil, nil)
	var respObj runtime.Object
	switch *requestGvk {
	case v12.SchemeGroupVersion.WithKind("AdmissionReview"):
		ar, ok := obj.(*v12.AdmissionReview)
		if !ok {
			glog.Error(errors.Wrap(err, "error decoding obj into v1.AdmissionReview"))
			return
		}
		resp := &v12.AdmissionReview{}
		resp.Response = handlers[gvk].V1(r.Context(), ar.Request)
		resp.SetGroupVersionKind(*requestGvk)
		resp.Response.UID = ar.Request.UID
		respObj = resp
	case v1beta1.SchemeGroupVersion.WithKind("AdmissionReview"):
		ar, ok := obj.(*v1beta1.AdmissionReview)
		if !ok {
			glog.Error(errors.Wrap(err, "error decoding obj into v1beta1.AdmissionReview"))
			return
		}
		resp := &v1beta1.AdmissionReview{}
		resp.Response = handlers[gvk].V1beta1(r.Context(), ar.Request)
		resp.SetGroupVersionKind(*requestGvk)
		resp.Response.UID = ar.Request.UID
		respObj = resp
	default:
		glog.Errorf("invalid gvk passed to admission review webhook: %s", requestGvk)
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "invalid gvk")
		return
	}

	encoder := json.NewEncoder(w)
	w.Header().Add("Content-Type", "application/json")
	if err := encoder.Encode(respObj); err != nil {
		glog.Error(errors.Wrap(err, "error encoding admissionreview response"))
	}
}