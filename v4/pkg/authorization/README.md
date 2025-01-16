## Authorization

Can a user perform an action? This package determines that. 

Based on roles and rolebindings, as well as hard-coded superusers, this 
package determines if an action can be taken. 

A single method, `Authorize()`, is called by the apiserver for *every*
api request. 

This method looks up roles for a user (via rolebindings) and determines if those 
roles allow the action the user is attempting to take. 

A shortcut exists in the form of superusers. The purposes for these
superusers varies, but they are all core processes to HobbyFarm itself.
(for example the controller-manager)