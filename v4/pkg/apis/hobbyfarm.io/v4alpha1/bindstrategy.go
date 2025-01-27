package v4alpha1

type BindStrategy string

const (
	BindStrategyAnyAvailable       BindStrategy = "Any"
	BindStrategyPreferMachineSets  BindStrategy = "PreferMachineSets"
	BindStrategyRequireMachineSets BindStrategy = "RequireMachineSets"
)
