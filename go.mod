module github.com/hobbyfarm/gargantua

go 1.16

replace k8s.io/client-go => k8s.io/client-go v0.23.2

require (
	github.com/dgrijalva/jwt-go v3.2.1-0.20200107013213-dc14462fd587+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.3
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rancher/k3d/v5 v5.3.0
	github.com/rancher/terraform-controller v0.0.10-alpha1
	github.com/rancher/wrangler v0.8.10
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	k8s.io/api v0.23.2
	k8s.io/apimachinery v0.23.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.23.2
	k8s.io/klog v1.0.0
)
