package rbac

const (
	OperatorAnd = "AND"
	OperatorOr = "OR"
)

type Request struct {
	operator string
	permissions []Permission
}

func RbacRequest() *Request {
	return &Request{
		permissions: []Permission{},
	}
}

func (r *Request) GetOperator() string {
	if r.operator == "" {
		return OperatorAnd
	}

	return r.operator
}

func (r *Request) GetPermissions() []Permission {
	return r.permissions
}

func (r *Request) And() *Request {
	r.operator = OperatorAnd
	return r
}

func (r *Request) Or() *Request {
	r.operator = OperatorOr
	return r
}

func (r *Request) HobbyfarmPermission(resource string, verb string) *Request {
	r.permissions = append(r.permissions, HobbyfarmPermission{Resource: resource, Verb: verb})
	return r
}

func (r *Request) Permission(apigroup string, resource string, verb string) *Request {
	r.permissions = append(r.permissions, GenericPermission{APIGroup: apigroup, Resource: resource, Verb: verb})
	return r
}

type Permission interface {
	GetAPIGroup() string
	GetResource() string
	GetVerb() string
}

type GenericPermission struct {
	APIGroup string
	Resource string
	Verb string
}

type HobbyfarmPermission struct {
	Resource string
	Verb string
}

func (g GenericPermission) GetAPIGroup() string {
	return g.APIGroup
}

func (g GenericPermission) GetResource() string {
	return g.Resource
}

func (g GenericPermission) GetVerb() string {
	return g.Verb
}

func (h HobbyfarmPermission) GetAPIGroup() string {
	return APIGroup
}

func (h HobbyfarmPermission) GetResource() string {
	return h.Resource
}

func (h HobbyfarmPermission) GetVerb() string {
	return h.Verb
}