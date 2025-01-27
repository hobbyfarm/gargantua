## pkg/apis/hobbyfarm.io

This is the top-level directory that contains the type definitions 
for HobbyFarm. 

Definitions are stored in versioned folders, for example `v4alpha1/`

Types *may* be split up into individual go files 

## Core types
Each type has the following:
1. A declaration for the type itself, of the form `type Foo struct`
   2. An annotation on this for code generation purposes
      3. `// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object`
2. A declaration for the list type of the object, of the form `type FooList struct`
   3. An annotation on this for code generation purposes
      4. `// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object`

Types optionally may have a `Spec` or `Status` type associated with them.
If so, this is typically called `FooSpec` or `FooStatus`, and is added to the 
type via `Spec` and `Status` fields respectively. 

All fields on all types must have JSON struct tags for (de)serializing purposes.

All types also must have a method called `NamespaceScoped()` which is used in
apiserver internals to denote that the types are (not) namespaced.

## Additional types

There are types defined in these packages that do not conform to the standards
laid out in Core types. These are typically structs or string constants that are 
used as fields on the Core types but since they are not being individually 
handled by the apiserver they do not require the same "pomp and circumstance" 
as you see described above. 

An example of this is `v4alpha1/bindstrategy.go` or `v4alpha1/availabilityconfiguration.go`