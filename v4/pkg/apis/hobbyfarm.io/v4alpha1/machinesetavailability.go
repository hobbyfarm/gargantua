package v4alpha1

type MachineSetAvailability string

const (
	MachineSetAvailabilityAccessCode     MachineSetAvailability = "AccessCode"
	MachineSetAvailabilityScheduledEvent MachineSetAvailability = "ScheduledEvent"
	MachineSetAvailabilityPool           MachineSetAvailability = "Pool"
)
