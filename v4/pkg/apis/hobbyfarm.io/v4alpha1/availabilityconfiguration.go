package v4alpha1

type AvailabilityConfiguration struct {
	// Availability defines what strategy will be used for making machines available to users.
	Availability MachineSetAvailability `json:"availability"`

	// Value defines a string identifier related to the Availability. For example,
	// in the case of ScheduledEvent availability, this value may be the name of
	// the associated ScheduledEvent.
	Value string `json:"value,omitempty"`
}
