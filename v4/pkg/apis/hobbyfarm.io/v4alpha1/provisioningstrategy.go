package v4alpha1

type ProvisioningStrategy string

const (
	ProvisioningStrategyAutoScale ProvisioningStrategy = "AutoScale"
	ProvisioningStrategyDynamic   ProvisioningStrategy = "OnDemand"
)
