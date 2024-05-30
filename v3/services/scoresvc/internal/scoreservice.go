package scoreservice

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sort"
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

type LanguageLeaderboardWithLocalScores struct {
	Language    string  `json:"language"`
	Scores      []Score `json:"scores"`
	LocalScores []Score `json:"localscores"`
	Placement   int     `json:"placement"`
}

func (s *ScoreServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	language := mux.Vars(r)["language"] // Get language from URL parameter

	// If not found return empty leaderboard
	leaderboard := LanguageLeaderboard{
		Language: language,
		Scores:   []Score{},
	}

	leaderboardInterface, found := s.Cache.Get(language)
	if found {
		glog.Infof("Leaderboard not found: %s", language)
		// Type assertion
		lbType, ok := leaderboardInterface.(LanguageLeaderboard)
		if ok {
			leaderboard = s.rangeScores(lbType, 0, 10)
		}
	}

	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false) // This disables the escaping

	err := encoder.Encode(leaderboard)

	if err != nil {
		glog.Infof("Error marshalling leaderboard: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", buffer.Bytes())
}

func (s *ScoreServer) AddScoreFunc(w http.ResponseWriter, r *http.Request) {
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

	s.Mutex.Lock()
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

	s.Mutex.Unlock()

	leaderboardWithLocalScores := s.findLocalScores(temp.(LanguageLeaderboard), newScore)

	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false) // This disables the escaping

	err = encoder.Encode(leaderboardWithLocalScores)

	if err != nil {
		glog.Infof("Error marshalling leaderboard with local scores: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	util.ReturnHTTPContent(w, r, 200, "success", buffer.Bytes())
}

func (s *ScoreServer) ScanFunc(w http.ResponseWriter, r *http.Request) {
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
			// First decode from base64
		}

		cooldown := Cooldown{
			Cooldown: time.Now(),
		}

		responseData, err := json.Marshal(cooldown)
		if err != nil {
			glog.Infof("Error marshalling cooldown data: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		util.ReturnHTTPContent(w, r, 200, "ok", responseData)

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
func (s *ScoreServer) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Just to see of the service is up and running
func (s *ScoreServer) GetTimeout() time.Duration {
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

// top10Scores returns a LanguageLeaderboard with only the top 10 scores, ranked by score in descending order.
func (s *ScoreServer) rangeScores(leaderboard LanguageLeaderboard, offset int, limit int) LanguageLeaderboard {
	// Sort the Scores slice based on the Score field, in descending order.
	sort.Slice(leaderboard.Scores, func(i, j int) bool {
		return leaderboard.Scores[i].Score > leaderboard.Scores[j].Score
	})

	if offset > len(leaderboard.Scores) {
		leaderboard.Scores = []Score{} // Return an empty slice if offset is beyond the available scores
	} else {
		end := offset + limit
		if end > len(leaderboard.Scores) {
			end = len(leaderboard.Scores)
		}

		leaderboard.Scores = leaderboard.Scores[offset:end] // Select only the top 10 scores
	}

	return leaderboard
}

func (s *ScoreServer) findLocalScores(leaderboard LanguageLeaderboard, newScore Score) LanguageLeaderboardWithLocalScores {
	// Append the new score to the list temporarily for sorting and finding its position
	scores := append([]Score{}, leaderboard.Scores...) // Make a copy to avoid modifying original
	localScores := make([]Score, 0, 4)                 // slice to hold the local group
	scores = append(scores, newScore)

	// Sort scores by the score field
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// Find index of the new score
	index := sort.Search(len(scores), func(i int) bool {
		return scores[i].Score < newScore.Score
	}) - 1

	// We only need local scores if new score is not in the top 10

	if index >= 10 {
		// Get two scores above (if available)
		for i := index - 1; i >= 0 && i >= index-2 && i >= 10; i-- {
			localScores = append(localScores, scores[i])
		}
		// Get two scores below (if available)
		for i := index + 1; i < len(scores) && len(localScores) < 4; i++ {
			localScores = append(localScores, scores[i])
		}
	}

	top10Scores := s.rangeScores(leaderboard, 0, 10)

	return LanguageLeaderboardWithLocalScores{
		Language:    leaderboard.Language,
		Scores:      top10Scores.Scores,
		LocalScores: localScores,
		Placement:   index,
	}
}
