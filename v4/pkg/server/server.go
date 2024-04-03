package server

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/openapi/hobbyfarm_io"
	"github.com/hobbyfarm/gargantua/v4/pkg/scheme"
	"github.com/hobbyfarm/gargantua/v4/pkg/stores/kubernetes"
	"github.com/hobbyfarm/mink/pkg/server"
	apiServer "k8s.io/apiserver/pkg/server"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewKubernetesServer(client client.WithWatch) (*server.Server, error) {
	storage, err := kubernetes.NewKubernetesStorage(client)
	if err != nil {
		return nil, err
	}
	return server.New(&server.Config{
		Name:                         "hobbyfarm-api",
		Version:                      "v4alpha1",
		HTTPListenPort:               8080,
		HTTPSListenPort:              8443,
		LongRunningVerbs:             []string{"watch"},
		LongRunningResources:         nil,
		Scheme:                       scheme.Scheme,
		CodecFactory:                 &scheme.Codec,
		DefaultOptions:               nil,
		AuditConfig:                  nil,
		SkipInClusterLookup:          true,
		RemoteKubeConfigFileOptional: true,
		IgnoreStartFailure:           false,
		Middleware:                   nil,
		Authenticator:                nil,
		Authorization:                nil,
		OpenAPIConfig:                hobbyfarm_io.GetOpenAPIDefinitions,
		APIGroups: []*apiServer.APIGroupInfo{
			storage,
		},
		PostStartFunc:         nil,
		SupportAPIAggregation: false,
		ReadinessCheckers:     nil,
	})
}
