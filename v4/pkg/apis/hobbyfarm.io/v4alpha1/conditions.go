package v4alpha1

import "github.com/rancher/wrangler/pkg/condition"

const (
	ConditionActive   = condition.Cond("Active")
	ConditionInactive = condition.Cond("Inactive")
	ConditionPaused   = condition.Cond("Paused")
)
