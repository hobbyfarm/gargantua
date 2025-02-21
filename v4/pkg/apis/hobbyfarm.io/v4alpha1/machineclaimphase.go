package v4alpha1

type MachineClaimPhase string

const (
	MachineClaimPhaseRequested  MachineClaimPhase = "Requested"
	MachineClaimPhaseBound      MachineClaimPhase = "Bound"
	MachineClaimPhaseFailed     MachineClaimPhase = "Failed"
	MachineClaimPhaseTerminated MachineClaimPhase = "Terminated"
)
