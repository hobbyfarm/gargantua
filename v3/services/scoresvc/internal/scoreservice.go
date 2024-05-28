package scoreservice

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
)

type Score struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type LanguageLeaderboard struct {
	Language string  `json:"language"`
	Scores   []Score `json:"scores"`
}

func (s ScoreServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	language := mux.Vars(r)["language"] // Get language from URL parameter

	leaderboard, found := s.Cache.Get(language)
	if !found {
		glog.Infof("Leaderboard not found: %s", language)

		// If not found return empty leaderboard
		leaderboard = LanguageLeaderboard{
			Language: language,
			Scores:   []Score{},
		}
	}

	responseData, err := json.Marshal(leaderboard)
	if err != nil {
		glog.Infof("Error marshalling leaderboard: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", responseData)
}

func (s ScoreServer) AddScoreFunc(w http.ResponseWriter, r *http.Request) {
	var newScore Score
	err := json.NewDecoder(r.Body).Decode(&newScore)
	if err != nil {
		glog.Infof("Error decoding score: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	language := mux.Vars(r)["language"] // Get language from URL parameter

	// Retrieve the existing leaderboard from cache
	temp, found := s.Cache.Get(language)
	if !found {
		// If not found, initialize a new leaderboard
		temp = LanguageLeaderboard{
			Language: language,
			Scores:   []Score{},
		}
	}

	leaderboard := temp.(LanguageLeaderboard)
	leaderboard.Scores = append(leaderboard.Scores, newScore)

	// Update the cache with the new leaderboard
	s.Cache.Set(language, leaderboard, 0)

	w.WriteHeader(http.StatusCreated)
}

// Just to see of the service is up and running
func (s ScoreServer) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
