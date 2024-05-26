package scoreservice

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	cache "github.com/patrickmn/go-cache"
)

type ScoreServer struct {
	Cache *cache.Cache
}

func NewScoreServer() (ScoreServer, error) {
	s := ScoreServer{}
	s.Cache = cache.New(cache.NoExpiration, cache.NoExpiration)
	return s, nil
}

func (s ScoreServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/score/leaderboard/{language}", s.GetFunc).Methods("GET")
	r.HandleFunc("/score/add/{language}", s.AddScoreFunc).Methods("POST")
	glog.V(2).Infof("set up routes for Score server")
}
