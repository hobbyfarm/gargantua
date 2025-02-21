## names

Names is a package very similar to `labels`. However, instead of defining labels that 
are used on objects, `names` defines the _names_ of those objects. 

This is useful for example with settings. 

HobbyFarm uses settings to control its behavior. Because of this, it needs to be able
to refer to these settings. `names` is where we define the names of these settings 
so they are static across all areas of the application. 