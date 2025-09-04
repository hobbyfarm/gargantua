package quiz

import (
	"strings"
)

type PreparedQuiz struct {
	Id               string             `json:"id,omitempty"`
	Title            string             `json:"title"`
	Issuer           string             `json:"issuer"`
	Shuffle          bool               `json:"shuffle"`
	PoolSize         uint32             `json:"pool_size"`
	MaxAttempts      uint32             `json:"max_attempts"`
	SuccessThreshold uint32             `json:"success_threshold"`
	ValidationType   string             `json:"validation_type"`
	Questions        []PreparedQuestion `json:"questions"`
}

type PreparedQuestion struct {
	Id             string           `json:"id,omitempty"`
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Type           string           `json:"type"`
	Shuffle        bool             `json:"shuffle"`
	FailureMessage string           `json:"failure_message"`
	SuccessMessage string           `json:"success_message"`
	Weight         uint32           `json:"weight"`
	Answers        []PreparedAnswer `json:"answers"`
}

type PreparedAnswer struct {
	Id      string `json:"id,omitempty"`
	Title   string `json:"title"`
	Correct *bool  `json:"correct,omitempty"`
}

type ValidationType = string

const (
	ValidationTypeDetailed ValidationType = "detailed"
	ValidationTypeStandard ValidationType = "standard"
	ValidationTypeNone     ValidationType = "none"
)

func parseValidationType(s string) ValidationType {
	lower := strings.ToLower(s)
	switch lower {
	case ValidationTypeNone:
		return ValidationTypeNone
	case ValidationTypeStandard:
		return ValidationTypeStandard
	case ValidationTypeDetailed:
		return ValidationTypeDetailed
	default:
		return ValidationTypeNone
	}
}
