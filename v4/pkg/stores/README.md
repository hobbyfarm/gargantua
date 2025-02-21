## stores

In this package is defined all the logic for storing our HobbyFarm resources in a 
remote (host) k8s cluster or in a SQL database (tbd). 

Five main subdirectories.

### kubernetes

In `kubernetes/` we wire up what are called "Remotes". Remotes are clients that interact
with a remote k8s server and perform storage actions for us. Remotes are how 
HobbyFarm API calls for read, or update, translate into k8s calls of POST, or GET. 

### registry

In `registry/` we build the actual storage structs that are called by our apiserver 
for any action. 

There is a main method in each file, `NewFooStorage` which takes as an argument
a "Strategy". A strategy is an interface whose implementation defines how resources
are stored in a storage of some kind (remote k8s, sql, etc.). 

Thus, when building remote storage of resources in a host k8s cluster, we call
`NewFooStorage` and pass it a Remote that we built in `kubernetes/`. In fact, that's
exactly what happens in `v4/server` code: 

```go
machineTemplateStorage, err := registry.NewMachineTemplateStorage(storages["machinetemplates"])
```

`NewMachineTemplateStorage` is a storage struct, and the value at 
`storages["machinetemplates"]` is the kubernetes remote. 

### remote

In `remote` is a struct called `NamespaceScopedRemote`. This is a helper struct that wraps
a k8s client and enforces a specific namespace for all actions performed. 

In other words, when you typically call a k8s client you can specify your namespace of choice. 

With `NamespaceScopedRemote` you are forced into whatever namespace has been defined. 

This is important because we don't want to leak our resources into another namespace in our 
host k8s cluster. We only want them to be in whatever namespace we have defined. 

Basically it saves ourselves from a footgun. 

### strategy

Strategies as discussed under `registry/` are interfaces that define methods 
we use to store (and retrieve, and update, and delete) resources. 

### translators

In this package we define translation methods.

Translation methods are helper methods that translate resources stored on our host k8s cluster
into resources that HobbyFarm can manipulate. 

An example is a ConfigMap. 

HobbyFarm has its own definition of a ConfigMap. However in the underlying (host)
k8s cluster, we store these as regular old k8s ConfigMaps. 

We can't directly turn a `corev1.ConfigMap` into `v4alpha1.ConfigMap` because
Go just doesn't work that way. 

So we need to define translators that handle the conversion for us. 