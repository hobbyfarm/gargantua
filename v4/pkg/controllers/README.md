## controllers

The core controllers that comprise most HobbyFarm control loops are stored here. 

These controllers are built in the following manner:

1. Each package has a top-level method called `RegisterHandlers()` to which
a `github.com/rancher/lasso/pkg/controller/SharedControllerFactory` is passed.
   2. The factory is used to create controllers for any desired type, e.g.
   `v4alpha1.User`. 
   3. The factory is also used to create k8s clients for any desired type. These
   clients point to the HF apiserver (not k8s) which is important to remember. 
   4. The factory is also used to create caches (and indexes on those caches) which
   are efficient methods of lookup for computationally expensive-to-get resources.
5. In the `RegisterHandlers()` method, once a controller for a type has been created, 
you may register handlers on that controller. These are the control loops that run for
every object of that kind. 

If you are only registering a single handler which only manipulates that specific object
(i.e. you don't need to retrieve/manipulate other resources), you can create a non-receiver
method and register it using `controller.SharedControllerHandlerFunc` from rancher/lasso.

If you are making a more complex controller it is recommended to create a struct to store
your clients, indexers, caches, etc. that you may need. You can then register your handlers
using `controller.SharedControllerHandlerFunc` again, and passing the instance
of your struct and the func that hangs off of it, e.g. 
`controller.SharedControllerHandlerFunc(mycx.handleStuff)`

