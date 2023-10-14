package rbac

// operators
const (
	OperatorAND = "AND"
	OperatorOR  = "OR"
)

// verbs
const (
	VerbList   = "list"
	VerbGet    = "get"
	VerbCreate = "create"
	VerbUpdate = "update"
	VerbDelete = "delete"
	VerbWatch  = "watch"
)

// group(s)
const (
	HobbyfarmGroup = "hobbyfarm.io"
	RbacGroup      = "rbac.authorization.k8s.io"
)

// resource plurals
const (
	ResourcePluralCourse      = "courses"
	ResourcePluralScenario    = "scenarios"
	ResourcePluralUser        = "users"
	ResourcePluralSession     = "sessions"
	ResourcePluralVM          = "virtualmachines"
	ResourcePluralVMSet       = "virtualmachinesets"
	ResourcePluralVMClaim     = "virtualmachineclaims"
	ResourcePluralVMTemplate  = "virtualmachinetemplates"
	ResourcePluralEnvironment = "environments"
	ResourcePluralEvent       = "scheduledevents"
	ResourcePluralProgress    = "progresses"
	ResourcePluralRole        = "roles"
	ResourcePluralRolebinding = "rolebindings"
	ResourcePluralOTAC        = "onetimeaccesscodes"
	ResourcePluralSettings    = "settings"
	ResourcePluralScopes      = "scopes"
)
