module github.com/hobbyfarm/gargantua

go 1.13

replace k8s.io/client-go => k8s.io/client-go v0.15.8

require (
	github.com/dgrijalva/jwt-go v3.2.1-0.20200107013213-dc14462fd587+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/mux v1.7.1
	github.com/gorilla/websocket v1.4.0
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rancher/terraform-controller v0.0.10-alpha1
	github.com/rancher/wrangler v0.1.0
	golang.org/x/crypto v0.0.0-20191227163750-53104e6ec876
	k8s.io/api v0.15.8
	k8s.io/apimachinery v0.15.8
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.15.8
	k8s.io/klog v1.0.0
)
