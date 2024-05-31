package scoreservice

import (
	"sync"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	cache "github.com/patrickmn/go-cache"
)

type ScoreServer struct {
	Cache *cache.Cache
	Mutex sync.Mutex
}

func NewScoreServer() (*ScoreServer, error) {
	s := ScoreServer{}
	s.Cache = cache.New(cache.NoExpiration, cache.NoExpiration)
	return &s, nil
}

func (s *ScoreServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/score/leaderboard/{language}", s.GetFunc).Methods("GET")
	r.HandleFunc("/score/add/{language}", s.AddScoreFunc).Methods("POST")
	r.HandleFunc("/score/scan/{code}", s.ScanFunc).Methods("POST")
	r.HandleFunc("/score/qrcode/{code}", s.HandleGenerateQR).Methods("GET")
	r.HandleFunc("/score/healthz", s.Healthz).Methods("GET")
	glog.V(2).Infof("set up routes for Score server")
}
