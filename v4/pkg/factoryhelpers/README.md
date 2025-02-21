## factoryhelpers

This package exists to make easier the work of generating certain objects when using
controllers. 

As of this writing there is a single method, `ClientForObject`, which retrieves a 
lasso `Client` for a given object. 

This is used as a shortcut to get a k8s client for an object when building a controller.  