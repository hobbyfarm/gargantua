package scoreservice

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
)

const (
	DEFAULT_COOLDOWN_DURATION = time.Hour * 1
)

type Cooldown struct {
	Cooldown time.Time `json:"cooldown"`
}

type Score struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Code  string `json:"code"`
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

	if newScore.Code != "" {
		// Check if this score.code is on cooldown
		codeCooldownCacheId := "scan_" + newScore.Code + "_cooldown"
		_, exp, found := s.Cache.GetWithExpiration(codeCooldownCacheId)

		if found {
			cooldown := Cooldown{
				Cooldown: exp,
			}

			responseData, err := json.Marshal(cooldown)
			if err != nil {
				glog.Infof("Error marshalling cooldown data: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			util.ReturnHTTPContent(w, r, 429, "oncooldown", responseData)
			return
		}
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
	s.Cache.Set(language, leaderboard, -1)

	w.WriteHeader(http.StatusCreated)
}

func (s ScoreServer) ScanFunc(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"] // Get language from URL parameter

	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code = "scan_" + code
	scanner_cooldown_id := code + "_cooldown"

	// Retrieve existing scan from cache
	_, expiration, found := s.Cache.GetWithExpiration(scanner_cooldown_id)
	if !found {
		s.Cache.Set(scanner_cooldown_id, true, s.GetTimeout())

		_, found := s.Cache.Get(code)
		if !found {
			s.Cache.Set(code, true, -1)
			// TODO send code to ms teams
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	cooldown := Cooldown{
		Cooldown: expiration,
	}

	responseData, err := json.Marshal(cooldown)
	if err != nil {
		glog.Infof("Error marshalling cooldown data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	util.ReturnHTTPContent(w, r, 429, "oncooldown", responseData)

}

// Just to see of the service is up and running
func (s ScoreServer) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Just to see of the service is up and running
func (s ScoreServer) GetTimeout() time.Duration {
	timeout, found := os.LookupEnv("COOLDOWN_DURATION")
	if !found {
		return DEFAULT_COOLDOWN_DURATION
	}

	timeoutDurationWithDays, err := util.GetDurationWithDays(timeout)
	if err != nil {
		return DEFAULT_COOLDOWN_DURATION
	}

	timeoutDuration, err := time.ParseDuration(timeoutDurationWithDays)
	if err != nil {
		return DEFAULT_COOLDOWN_DURATION
	}

	return timeoutDuration
}
