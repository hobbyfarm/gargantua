package shell

import (
	"fmt"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	"github.com/hobbyfarm/gargantua/pkg/rbacclient"
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

	hfClient   hfClientset.Interface
	kubeClient kubernetes.Interface
	ctx        context.Context
}

var sshDev = ""
var sshDevHost = ""
var sshDevPort = ""
var guacHost = ""
var guacPort = ""

const (
	resourcePlural      = "virtualmachines"
	defaultSshUsername = "ubuntu"
)

// SIGWINCH is the regex to match window change (resize) codes
var SIGWINCH *regexp.Regexp
var sess *ssh.Session

var DefaultDialer = websocket.DefaultDialer

func init() {
	sshDev = os.Getenv("SSH_DEV")
	sshDevHost = os.Getenv("SSH_DEV_HOST")
	sshDevPort = os.Getenv("SSH_DEV_PORT")
	guacHost = os.Getenv("GUAC_SERVICE_HOST") //Get the Guac Host. This is set by kubernetes
	guacPort = os.Getenv("GUAC_SERVICE_PORT") //Get the Guac Port. This is set by kubernetes
	SIGWINCH = regexp.MustCompile(`.*\[8;(.*);(.*)t`)
}

func NewShellProxy(authClient *authclient.AuthClient, vmClient *vmclient.VirtualMachineClient, hfClientSet hfClientset.Interface, kubeClient kubernetes.Interface, ctx context.Context) (*ShellProxy, error) {
	shellProxy := ShellProxy{}

	shellProxy.auth = authClient
	shellProxy.vmClient = vmClient
	shellProxy.hfClient = hfClientSet
	shellProxy.kubeClient = kubeClient
	shellProxy.ctx = ctx

	return &shellProxy, nil
}

func (sp ShellProxy) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/shell/{vm_id}/connect", sp.ConnectSSHFunc)
	r.HandleFunc("/guacShell/{vm_id}/connect", sp.ConnectGuacFunc)
	glog.V(2).Infof("set up routes")
}

/*
* This is used for all connections made via the guacamole client
* Currently supported protocols are: rdp, vnc, telnet, ssh
 */
func (sp ShellProxy) ConnectGuacFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sp.auth.AuthWS(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}

	vars := mux.Vars(r)

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

	glog.Infof("Going to upgrade guac connection now... %s", vm.Spec.Id)

	// ok first get the secret for the vm
	secret, err := sp.kubeClient.CoreV1().Secrets(util.GetReleaseNamespace()).Get(sp.ctx, vm.Spec.SecretName, v1.GetOptions{}) // idk?
	if err != nil {
		glog.Errorf("did not find secret for virtual machine")
		util.ReturnHTTPMessage(w, r, 500, "error", "unable to find keypair secret for vm")
		return
	}

	password := string(secret.Data["password"])

	username := vm.Spec.SshUsername
	if len(username) < 1 {
		username = defaultSshUsername
	}

	// get the host and port
	host := vm.Status.PublicIP
	protocol := strings.ToLower(vm.Spec.Protocol)
	port := mapProtocolToPort()[protocol]

	optimalHeight := r.URL.Query().Get("height")
	optimalWidth := r.URL.Query().Get("width")

	//
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// todo - HACK
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	// GoogleChrome needs the Sec-Websocket-Protocol Header be set to the requested protocol
	ws_protocol := r.Header.Get("Sec-Websocket-Protocol")
	conn, err := upgrader.Upgrade(w, r, http.Header{
		"Sec-Websocket-Protocol": {ws_protocol},
	}) // upgrade to websocket

	if err != nil {
		glog.Errorf("error upgrading: %s", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error upgrading to websocket")
		return
	}
	defer conn.Close()

	backendURL := fmt.Sprintf("ws://%s:%s/websocket-tunnel", guacHost, guacPort)
	requestHeader := http.Header{}

	//Use url.query, as this provides a query.Encode() method.
	u, _ := url.Parse("http://example.com") //just to get u.Query()
	q := u.Query()
	q.Set("scheme", protocol)
	q.Set("hostname", host)
	q.Set("port", strconv.Itoa(port))
	q.Set("ignore-cert", "true")
	q.Set("username", username)
	q.Set("password", password)
	q.Set("width", optimalWidth)
	q.Set("height", optimalHeight)
	q.Set("security", "")

	backendURL += "?" + q.Encode()
	//Replace to keep the password out of the logs! Replacing "password=<password>" instead of only "<password>", for cases where the password is short and/or is contained in other parameters
	glog.V(6).Infof("Build query " + strings.Replace(backendURL, "password="+password, "password=XXX_PASSWORD_XXX", 1))

	connBackend, resp, err := DefaultDialer.Dial(backendURL, requestHeader)
	if err != nil {
		glog.Errorf("websocketproxy: couldn't dial to remote backend url %s", err)
		if resp != nil {
			// If the WebSocket handshake fails, ErrBadHandshake is returned
			// along with a non-nil *http.Response so that callers can handle
			// redirects, authentication, etcetera.
			if err := copyResponse(w, resp); err != nil {
				glog.Errorf("websocketproxy: couldn't write response after failed remote backend handshake: %s", err)
			}
		} else {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
		return
	}
	defer connBackend.Close()

	errClient := make(chan error, 1)
	errBackend := make(chan error, 1)
	replicateWebsocketConn := func(dst, src *websocket.Conn, errc chan error) {
		for {
			msgType, msg, err := src.ReadMessage()
			if err != nil {
				m := websocket.FormatCloseMessage(websocket.CloseNormalClosure, fmt.Sprintf("%v", err))
				if e, ok := err.(*websocket.CloseError); ok {
					if e.Code != websocket.CloseNoStatusReceived {
						m = websocket.FormatCloseMessage(e.Code, e.Text)
					}
				}
				errc <- err
				dst.WriteMessage(websocket.CloseMessage, m)
				break
			}
			err = dst.WriteMessage(msgType, msg)
			if err != nil {
				errc <- err
				break
			}
		}
	}

	go replicateWebsocketConn(conn, connBackend, errClient)
	go replicateWebsocketConn(connBackend, conn, errBackend)

	var message string
	select {
	case err = <-errClient:
		message = "websocketproxy: Error when copying from backend to client: %v"
	case err = <-errBackend:
		message = "websocketproxy: Error when copying from client to backend: %v"

	}
	if e, ok := err.(*websocket.CloseError); !ok || e.Code == websocket.CloseAbnormalClosure {
		glog.Errorf(message, err)
	}

}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func copyResponse(rw http.ResponseWriter, resp *http.Response) error {
	copyHeader(rw.Header(), resp.Header)
	rw.WriteHeader(resp.StatusCode)
	defer resp.Body.Close()

	_, err := io.Copy(rw, resp.Body)
	return err
}

/*
* This is mainly used for SSH Connections to VMs
 */
func (sp ShellProxy) ConnectSSHFunc(w http.ResponseWriter, r *http.Request) {
	user, err := sp.auth.AuthWS(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}

	vars := mux.Vars(r)

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
		// check if the user has access to access user sessions
		// TODO: add permission like 'virtualmachine/shell' similar to 'pod/exec'
		_, err := sp.auth.AuthGrant(
			rbacclient.RbacRequest().
				HobbyfarmPermission("users", rbacclient.VerbGet).
				HobbyfarmPermission("sessions", rbacclient.VerbGet).
				HobbyfarmPermission("virtualmachines", rbacclient.VerbGet),
			w, r)
		if err != nil {
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "access denied to connect to ssh shell session")
			return
		}
	}

	glog.Infof("Going to upgrade connection now... %s", vm.Spec.Id)

	// ok first get the secret for the vm
	secret, err := sp.kubeClient.CoreV1().Secrets(util.GetReleaseNamespace()).Get(sp.ctx, vm.Spec.SecretName, v1.GetOptions{}) // idk?
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

	sshUsername := vm.Spec.SshUsername
	if len(sshUsername) < 1 {
		sshUsername = defaultSshUsername
	}

	// now use the secret and ssh off to something
	config := &ssh.ClientConfig{
		User: sshUsername,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// get the host and port
	host, ok := vm.Annotations["sshEndpoint"]
	if !ok {
		host = vm.Status.PublicIP
	}
	port := "22"
	if sshDev == "true" {
		if sshDevHost != "" {
			host = sshDevHost
		}
		if sshDevPort != "" {
			port = sshDevPort
		}
	}

	// dial the instance
	sshConn, err := ssh.Dial("tcp", host+":"+port, config)
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

	sess, err = sshConn.NewSession()
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

	err = sess.RequestPty("xterm", 40, 80, ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400})
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

func mapProtocolToPort() map[string]int {
	m := make(map[string]int)
	m["rdp"] = 3389
	m["vnc"] = 5900
	m["telnet"] = 23
	m["ssh"] = 22
	return m
}

func ResizePty(h int, w int) {
	if err := sess.WindowChange(h, w); err != nil {
		glog.Warningf("error resizing pty: %s", err)
	}
}
