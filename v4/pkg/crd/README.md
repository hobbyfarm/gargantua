## crd

In order to store HobbyFarm types in the underlying (host) k8s cluster, they must be 
registered as custom resources with that cluster's apiserver. It is within this
package that those actions are taken. 

Every type that needs to be stored (i.e. things for which there are annotations in 
`apis/`) is listed here as a CRD. There are helper methods to make this easier. 

Review the following code:

```go
hobbyfarmCRD(&v4alpha1.MachineTemplate{}, func(c *crder.CRD) {
    c.
        IsNamespaced(true).
        AddVersion("v4alpha1", &v4alpha1.Provider{}, func(cv *crder.Version) {
            cv.
                WithColumn("Type", ".spec.machineType").
                WithColumn("DisplayName", ".spec.displayName").
                WithColumn("Protocols", ".spec.connectProtocol").
                WithColumn("Prefix", ".spec.machineNamePrefix")
        }).
        WithShortNames("mt")
}),
```

`hobbyfarmCRD()` is a method defined at the bottom of the file that sets up the CRD
to exist within the `hobbyfarm.io` group. Passed as a first argument to this method
is the type we wish to register, along with a customization function. That customization 
function takes a single argument, `*crder.CRD` which represents the CRD we are intending
to register. 

In that customization function we apply transformations to the `crder.CRD` such as 
identifying the CR as namespaced, or using `WithShortNames()` to define short names
we can use to lookup resources. 

Each `crder.CRD` requires *at least one* version be defined via the `AddVersion()` helper
method. That method takes as arguments the string name of the version, 
the Go type for that version, and another customization function for the version. 

The customization function allows us to apply transformations to the CRD that are 
*specific to the version we are defining*. For example in this code we are defining
output columns using the `WithColumn()` helper. These adjust how the output of 
`kubectl get machinetemplate` will look. 

At the end of registering all these CRDs, we return a slice of them all with
their customizations and versions. This is used elsewhere to generate the CRD manifests
that are then applied to the underlying k8s cluster on which HobbyFarm is running. 