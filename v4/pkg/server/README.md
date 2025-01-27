## server

This is where the core apiserver logic lives. 

This is where the bits from authentication, authorization, storage, 
etc. are combined and registered. 

This package heavily uses `github.com/hobbyfarm/mink` as the core engine
for the apiserver. 