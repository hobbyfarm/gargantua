package shell

import (
	"io"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"github.com/hobbyfarm/gargantua/pkg/vmclient"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ShellProxy struct {
	auth     *authclient.AuthClient
	vmClient *vmclient.VirtualMachineClient

	hfClient   *hfClientset.Clientset
	kubeClient *kubernetes.Clientset
}

func NewShellProxy(authClient *authclient.AuthClient, vmClient *vmclient.VirtualMachineClient, hfClientSet *hfClientset.Clientset, kubeClient *kubernetes.Clientset) (*ShellProxy, error) {
	shellProxy := ShellProxy{}

	shellProxy.auth = authClient
	shellProxy.vmClient = vmClient
	shellProxy.hfClient = hfClientSet
	shellProxy.kubeClient = kubeClient

	return &shellProxy, nil
}

func (sp ShellProxy) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/shell/{vm_id}/connect", sp.ConnectFunc)
	// registering twice because the width query param is optional
	r.HandleFunc("/shell/{vm_id}/connect", sp.ConnectFunc).Queries("width", "{width}")
	glog.V(2).Infof("set up routes")
}

func (sp ShellProxy) ConnectFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sp.auth.AuthWS(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}

	vars := mux.Vars(r)

	width := 80
	if r.FormValue("width") != "" {
		width, err = strconv.Atoi(r.FormValue("width"))
		if err != nil {
			util.ReturnHTTPMessage(w, r, 500, "error", "given width was not an integer")
			return
		}
	}

	vmId := vars["vm_id"]
	if len(vmId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm id passed in")
		return
	}

	vm, err := sp.vmClient.GetVirtualMachineById(vmId)

	if err != nil {
		glog.Errorf("did not find the right virtual machine ID")
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm found")
		return
	}

	if vm.Spec.UserId != user.Spec.Id {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "you do not have access to shell")
		return
	}

	glog.Infof("Going to upgrade connection now... %s", vm.Spec.Id)

	// ok first get the secret for the vm
	secret, err := sp.kubeClient.CoreV1().Secrets("hobbyfarm").Get(vm.Spec.KeyPair, v1.GetOptions{}) // idk?
	if err != nil {
		glog.Errorf("did not find secret for virtual machine")
		util.ReturnHTTPMessage(w, r, 500, "error", "unable to find keypair secret for vm")
		return
	}

	// parse the private key
	signer, err := ssh.ParsePrivateKey(secret.Data["private_key"])
	if err != nil {
		glog.Errorf("did not correctly parse private key")
		util.ReturnHTTPMessage(w, r, 500, "error", "unable to parse private key")
		return
	}

	// now use the secret and ssh off to something
	config := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// dial the instance
	sshConn, err := ssh.Dial("tcp", vm.Status.PublicIP+":22", config)
	if err != nil {
		glog.Errorf("did not connect ssh successfully: %s", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "could not establish ssh session to vm")
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// todo - HACK
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	conn, err := upgrader.Upgrade(w, r, nil) // upgrade to websocket
	if err != nil {
		glog.Errorf("error upgrading: %s", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error upgrading to websocket")
		return
	}

	wrapper := NewWSWrapper(conn, websocket.TextMessage)
	stdout := wrapper
	stderr := wrapper

	stdin := &InputWrapper{ws: conn}

	sess, err := sshConn.NewSession()
	if err != nil {
		glog.Errorf("did not setup ssh session properly")
		util.ReturnHTTPMessage(w, r, 500, "error", "could not setup ssh session")
		return
	}

	go func() {
		pip, _ := sess.StdoutPipe()
		io.Copy(stdout, pip)
	}()

	go func() {
		pip, _ := sess.StderrPipe()
		io.Copy(stderr, pip)
	}()

	go func() {
		pip, _ := sess.StdinPipe()
		io.Copy(pip, stdin)
	}()

	err = sess.RequestPty("xterm", 40, width, ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400})
	if err != nil {
		glog.Error(err)
	}
	err = sess.Shell()
	if err != nil {
		glog.Error(err)
	}

	//sess.Wait()
	//
	//defer sess.Close()
}
