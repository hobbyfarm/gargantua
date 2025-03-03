package main

import (
	"sync"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/pkg/signals"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"

	quizservice "github.com/hobbyfarm/gargantua/services/quizsvc/v3/internal"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	quizpb "github.com/hobbyfarm/gargantua/v3/protos/quiz"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	stopCh := signals.SetupSignalHandler()

	cfg, hfClient, _ := microservices.BuildClusterConfig(serviceConfig)

	namespace := util.GetReleaseNamespace()
	hfInformerFactory := hfInformers.NewSharedInformerFactoryWithOptions(hfClient, time.Second*30, hfInformers.WithNamespace(namespace))

	crd.InstallCrds(quizservice.QuizCRDInstaller{}, cfg, "quiz")

	services := []microservices.MicroService{
		microservices.AuthN,
		microservices.AuthR,
	}
	connections := microservices.EstablishConnections(services, serviceConfig.ClientCert)
	for _, conn := range connections {
		defer conn.Close()
	}
	authnClient := authnpb.NewAuthNClient(connections[microservices.AuthN])
	authrClient := authrpb.NewAuthRClient(connections[microservices.AuthR])

	gs := microservices.CreateGRPCServer(serviceConfig.ServerCert.Clone())

	qs := quizservice.NewGrpcQuizServer(hfClient, hfInformerFactory)
	quizpb.RegisterQuizSvcServer(gs, qs)

	var wg sync.WaitGroup
	// only add 1 to our wait group since our service should stop (and restart) as soon as one of the go routines terminates
	wg.Add(1)

	go func() {
		defer wg.Done()
		microservices.StartGRPCServer(gs, serviceConfig.EnableReflection)
	}()

	go func() {
		defer wg.Done()

		quizServer := quizservice.NewQuizServer(
			authnClient,
			authrClient,
			qs,
		)
		microservices.StartAPIServer(quizServer)
	}()

	hfInformerFactory.Start(stopCh)

	wg.Wait()
}
