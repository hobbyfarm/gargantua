## pkg/authentication

Everything in this package deals with authentication. 


### `authenticators/`
This package contains code necessary to authenticate incoming requests. 
This is done on a per-request basis and this code is invoked on every api 
request that is not exempted (e.g. /login, etc.)

As of this writing there are two types of authenticators, token and cert. 

Cert authentication takes incoming requests and pulls out the client key and 
client certificate from the request. These are then validated against the 
CA cert that the apiserver uses. 

Token authentication takes incoming requests and pulls out a JWT from
the `Authorization` header of the HTTP request.

In `authenticators/chain.go` each authentication method is tried in turn. 
If a method fails, we move onto the next one. If a method succeeds, we short-cut
and begin executing the request. 

This does *not* handle anything to do with authorization, only authN.

### `group/`

This package contains helpers for working with hobbyfarm Group objects. 

Specifically we need to be able to index Groups based on the members
of the group, which is important when authenticating users (see `providers/`). 

### `providers/`

This package handles logging in users. It is where various authentication providers
are defined, such as local authentication or ldap. 

Each provider has its own method of functionality, but must offer a `HandleLogin` 
method to handle authentication requests. Based on that request a provider
may perform different actions such as verifying group memberships, authenticating
against an outside source, etc. 

### `user/`

The user package defines a common struct that all providers can use when referring
to a user. It is also the implementation of a User interface that is necessary
for interaction with the underlying k8s components that make up the apiserver
(`GetName()`, `GetUID()`, `GetGroups()`)

At the top level is `authentication.go` which sets up all the auth providers 
and initializes caches for efficient lookups. 