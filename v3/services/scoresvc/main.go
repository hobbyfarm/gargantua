package main

import (
	"sync"

	scoreservice "github.com/hobbyfarm/gargantua/services/scoresvc/v3/internal"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"

	"github.com/golang/glog"
)

var (
	serviceConfig *microservices.ServiceConfig
)

func init() {
	serviceConfig = microservices.BuildServiceConfig()
}

func main() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		scoreServer, err := scoreservice.NewScoreServer()
		if err != nil {
			glog.Fatalf("Error creating userserver: %v", err)
		}
		microservices.StartAPIServer(scoreServer)
	}()

	wg.Wait()
}
