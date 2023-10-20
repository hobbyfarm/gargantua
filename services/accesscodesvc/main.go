package main

import (
	"context"
	"flag"
	"sync"

	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
	accesscodeservice "github.com/hobbyfarm/gargantua/services/accesscodesvc/v3/internal"
	accessCodeProto "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	flag.Parse()

	cfg, hfClient := microservices.BuildClusterConfig(serviceConfig)

	crds := accesscodeservice.GenerateAccessCodeCRD()
	glog.Info("installing/updating access code CRDs")
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating access code CRDs: %s", err.Error())
	}
	glog.Info("finished installing/updating access code CRDs")

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())
	ctx := context.Background()
	as := accesscodeservice.NewGrpcAccessCodeServer(hfClient, ctx)
	accessCodeProto.RegisterAccessCodeSvcServer(gs, as)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	wg.Wait()
}
