## certs

This package contains helpers for generating and signing certificates. 

For authentication to HobbyFarm, one may use certificates. 

In the case of core processes (such as controller-manager), certificates
are the method of authentication. These helpers are used to generate
those certs via the `cert-generator` service. 