package v4alpha1

type PauseBehavior string

const (
	// CanPause means a user CAN pause their course/scenario
	CanPause PauseBehavior = "CanPause"

	// CannotPause means a user CANNOT pause their course/scenario
	CannotPause PauseBehavior = "CannotPause"
)
