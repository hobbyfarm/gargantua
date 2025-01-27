package v4alpha1

type MachinePersistenceStrategy string

const (
	PersistThroughCourse MachinePersistenceStrategy = "PersistThroughCourse"
	NewPerScenario       MachinePersistenceStrategy = "NewPerScenario"
)
