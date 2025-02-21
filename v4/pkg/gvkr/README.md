## gvkr

GVKR defines a helper method for generating 
`GroupVersionKind` and `GroupVersionResource` objects for a given
set of strings for those types. 

### Examples
#### GroupVersionKind
Group: hobbyfarm.io

Version: v4alpha1

Kind: Environment

#### GroupVersionResource

Group: hobbyfarm.io

Version: v4alpha1

Kind: environments

---

GVK describes a type (Golang)
GVR is used for api interactions (e.g. `/hobbyfarm.io/v4alpha1/environments`)