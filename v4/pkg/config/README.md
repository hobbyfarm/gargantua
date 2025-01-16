## config

This package contains the configuration for the apiserver defined
as viper flags. 

This is where you can add additional flags to the apiserver or adjust
existing ones. 

An example flag is `--remote-k8s-namespace` which defines in which namespace
the HobbyFarm resources should be stored on the host k8s cluster. 

**As of this writing, these flags are not being used in the apiserver 
`main.go`. It is a future goal.**