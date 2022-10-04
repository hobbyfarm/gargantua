/*
Portions of this file are
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conversion

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/munnerz/goautoneg"
	"io/ioutil"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"net/http"
	"strings"
)

var converters = map[schema.GroupKind]conversionFunc{}

type conversionFunc func(unstructured *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status)

func New(mux *mux.Router, apiExtensionsClient *apiextensions.Clientset, caBundle string) {
	mux.HandleFunc("/conversion/{type}", dispatch).Methods(http.MethodPost)

	// for each conversion function, register the cabundle
	for gvk, _ := range converters {
		obj, err := apiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().Get(
			context.Background(), fmt.Sprintf("%s.%s", gvk.Kind, gvk.Group), metav1.GetOptions{})
		if err != nil {
			glog.Errorf("Error retrieving CustomResourceDefinition %s from api server: %s", gvk, err.Error())
			continue
		}

		obj = obj.DeepCopy()

		obj.Spec.Conversion.Webhook.ClientConfig.CABundle = []byte(caBundle)

		obj, err = apiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().Update(context.Background(), obj, metav1.UpdateOptions{})
		if err != nil {
			glog.Errorf("Error updating CRD %s with cabundle: %s", gvk, err.Error())
			continue
		}
	}
}

func RegisterConverter(gk schema.GroupKind, f conversionFunc) {
	converters[gk] = f
}

func doConversion(convertRequest *v1.ConversionRequest, convert conversionFunc) *v1.ConversionResponse {
	var convertedObjects []runtime.RawExtension
	for _, obj := range convertRequest.Objects {
		cr := unstructured.Unstructured{}
		if err := cr.UnmarshalJSON(obj.Raw); err != nil {
			glog.Error(err)
			return conversionResponseFailureWithMessagef("failed to unmarshall object (%v) with error: %v", string(obj.Raw), err)
		}
		convertedCR, status := convert(&cr, convertRequest.DesiredAPIVersion)
		if status.Status != metav1.StatusSuccess {
			glog.Error(status.String())
			return &v1.ConversionResponse{
				Result: status,
			}
		}
		convertedCR.SetAPIVersion(convertRequest.DesiredAPIVersion)
		convertedObjects = append(convertedObjects, runtime.RawExtension{Object: convertedCR})
	}
	return &v1.ConversionResponse{
		ConvertedObjects: convertedObjects,
		Result:           StatusSuccess(),
	}
}

func dispatch(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// lookup type for possible conversion
	objectType, ok := mux.Vars(r)["type"]
	if !ok || objectType == "" {
		http.Error(w, "invalid type for conversion", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	serializer := getInputSerializer(contentType)
	if serializer == nil {
		msg := fmt.Sprintf("invalid content-type header: %s", contentType)
		glog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	convertReview := v1.ConversionReview{}
	if _, _, err := serializer.Decode(body, nil, &convertReview); err != nil {
		glog.Error(err)
		convertReview.Response = conversionResponseFailureWithMessagef("failed to deserialize body (%v) with error %v", string(body), err)
	} else {
		convertReview.Response = doConversion(convertReview.Request, converters[parseGroupKind(objectType)])
		convertReview.Response.UID = convertReview.Request.UID
	}

	// reset the request, it is not needed in a response.
	convertReview.Request = &v1.ConversionRequest{}

	accept := r.Header.Get("Accept")
	outSerializer := getOutputSerializer(accept)
	if outSerializer == nil {
		msg := fmt.Sprintf("invalid accept header `%s`", accept)
		glog.Errorf(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	err := outSerializer.Encode(&convertReview, w)
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type mediaType struct {
	Type, SubType string
}

var scheme = runtime.NewScheme()
var serializers = map[mediaType]runtime.Serializer{
	{"application", "json"}: json.NewSerializer(json.DefaultMetaFactory, scheme, scheme, false),
	{"application", "yaml"}: json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme),
}

func getInputSerializer(contentType string) runtime.Serializer {
	parts := strings.SplitN(contentType, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	return serializers[mediaType{parts[0], parts[1]}]
}

func getOutputSerializer(accept string) runtime.Serializer {
	if len(accept) == 0 {
		return serializers[mediaType{"application", "json"}]
	}

	clauses := goautoneg.ParseAccept(accept)
	for _, clause := range clauses {
		for k, v := range serializers {
			switch {
			case clause.Type == k.Type && clause.SubType == k.SubType,
				clause.Type == k.Type && clause.SubType == "*",
				clause.Type == "*" && clause.SubType == "*":
				return v
			}
		}
	}

	return nil
}

func conversionResponseFailureWithMessagef(msg string, params ...interface{}) *v1.ConversionResponse {
	return &v1.ConversionResponse{
		Result: metav1.Status{
			Message: fmt.Sprintf(msg, params...),
			Status:  metav1.StatusFailure,
		},
	}

}

func StatusFailureWithMessage(msg string, params ...interface{}) metav1.Status {
	return metav1.Status{
		Message: fmt.Sprintf(msg, params...),
		Status:  metav1.StatusFailure,
	}
}

func StatusSuccess() metav1.Status {
	return metav1.Status{
		Status: metav1.StatusSuccess,
	}
}

func parseGroupKind(gk string) schema.GroupKind {
	gkSlice := strings.SplitN(gk, ".", 2)
	return schema.GroupKind{
		Group: gkSlice[1],
		Kind:  gkSlice[0],
	}
}
