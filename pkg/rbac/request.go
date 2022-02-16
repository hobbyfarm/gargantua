package rbac

type Request interface {
	GetAPIGroup() string
	GetResource() string
	GetVerb() string
}

type GenericRequest struct {
	APIGroup string
	Resource string
	Verb string
}

type HobbyfarmRequest struct {
	Resource string
	Verb string
}

func (g GenericRequest) GetAPIGroup() string {
	return g.APIGroup
}

func (g GenericRequest) GetResource() string {
	return g.Resource
}

func (g GenericRequest) GetVerb() string {
	return g.Verb
}

func (h HobbyfarmRequest) GetAPIGroup() string {
	return APIGroup
}

func (h HobbyfarmRequest) GetResource() string {
	return h.Resource
}

func (h HobbyfarmRequest) GetVerb() string {
	return h.Verb
}