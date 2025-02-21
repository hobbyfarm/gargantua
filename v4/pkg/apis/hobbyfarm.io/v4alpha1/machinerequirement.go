package v4alpha1

// MachineRequirement defines for a given MachineType and MachineTemplate, how many are required.
// This struct is split out from MachineProvisioningRequirement because it is re-used
// in contexts that do not require provisioning strategies, such as Course and Scenario.
type MachineRequirement struct {
	// MachineTemplate is the name of the required MachineTemplate
	MachineTemplate string `json:"machineTemplate"`

	// Count is the number of required Machines either per-User (when MachineType = MachineTypeUser)
	// or per- ScheduledEvent (when MachineType = MachineTypeShared)
	Count int `json:"count"`

	// MachineType is the type of machine to require
	MachineType MachineType `json:"machineType"`
}
