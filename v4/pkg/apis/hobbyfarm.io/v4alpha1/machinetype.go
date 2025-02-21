package v4alpha1

// MachineType defines what type of machine is defined in this template.
// For example, a machine may be MachineTypeUser which is a machine that can be assigned to a user.
// A MachineTypeShared may be accessible by multiple users.
type MachineType string

const (
	MachineTypeUser   MachineType = "User"
	MachineTypeShared MachineType = "Shared"
)
