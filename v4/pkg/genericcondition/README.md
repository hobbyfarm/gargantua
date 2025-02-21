## genericcondition

The genericcondition package defines a `GenericCondition` which is how types that require 
conditions in their Statuses record those conditions. 

It is named "generic" not because of Go generics but because it is not a condition
specific to any type or situation but is meant to encompass all possible.

You may notice annotations at the top of `genericcondition.go`:
```go
// +k8s:deepcopy-gen=package,register
// +k8s:openapi-gen=true
// +groupName=hobbyfarm.io
```

These are used when generating DeepCopy methods. 
Since generic conditions are stored in the status structs of some 
types, they need to be also able to be DeepCopied. 