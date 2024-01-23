package shell

import (
	"context"
	"encoding/json"
	"fmt"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	rbac2 "github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/pkg/vmclient"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	userProto "github.com/hobbyfarm/gargantua/v3/protos/user"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/semaphore"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ShellProxy struct {
	authnClient authn.AuthNClient
	authrClient authr.AuthRClient
	vmClient    *vmclient.VirtualMachineClient
	hfClient    hfClientset.Interface
	kubeClient  kubernetes.Interface
	ctx         context.Context
}

type Service struct {
	Name                string `json:"name"`
	HasWebinterface     bool   `json:"hasWebinterface"`
	Port                int    `json:"port"`
	Path                string `json:"path"`
	HasOwnTab           bool   `json:"hasOwnTab"`
	NoRewriteRootPath   bool   `json:"noRewriteRootPath"`
	RewriteHostHeader   bool   `json:"rewriteHostHeader"`
	RewriteOriginHeader bool   `json:"rewriteOriginHeader"`
}

var sshDev = ""
var sshDevHost = ""
var sshDevPort = ""
var guacHost = ""
var guacPort = ""

const (
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

func NewShellProxy(authnClient authn.AuthNClient, authrClient authr.AuthRClient, vmClient *vmclient.VirtualMachineClient, hfClientSet hfClientset.Interface, kubeClient kubernetes.Interface, ctx context.Context) (*ShellProxy, error) {
	shellProxy := ShellProxy{}

	shellProxy.authnClient = authnClient
	shellProxy.authrClient = authrClient
	shellProxy.vmClient = vmClient
	shellProxy.hfClient = hfClientSet
	shellProxy.kubeClient = kubeClient
	shellProxy.ctx = ctx

	return &shellProxy, nil
}

func (sp ShellProxy) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/shell/{vm_id}/connect", sp.ConnectSSHFunc)
	r.HandleFunc("/shell/verify", sp.VerifyTasksFuncByVMIdGroupWithSemaphore)
	r.HandleFunc("/guacShell/{vm_id}/connect", sp.ConnectGuacFunc)
	r.HandleFunc("/p/{vm_id}/{port}/{rest:.*}", sp.checkCookieAndProxy)
	r.HandleFunc("/pa/{token}/{vm_id}/{port}/{rest:.*}", sp.authAndProxyFunc)
	r.HandleFunc("/auth/{token}/{rest:.*}", sp.setAuthCookieAndRedirect)
	glog.V(2).Infof("set up routes")
}

func (sp ShellProxy) authAndProxyFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	authToken := vars["token"]
	user, err := sp.proxyAuth(w, r, authToken)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "invalid auth token")
		return
	}
	sp.proxy(w, r, user)
}

func (sp ShellProxy) setAuthCookieAndRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	authToken := vars["token"]
	_, err := sp.proxyAuth(w, r, authToken)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "invalid auth token")
		return
	}
	cookie := http.Cookie{Name: "jwt", Value: authToken, SameSite: http.SameSiteNoneMode, Secure: true, Path: "/"}
	http.SetCookie(w, &cookie)
	url := mux.Vars(r)["rest"]
	http.Redirect(w, r, "/"+url, 302)

}

/*
* Used to Proxy to Services exposed by the VM on specified Port
 */
func (sp ShellProxy) checkCookieAndProxy(w http.ResponseWriter, r *http.Request) {

	// Get the auth Variable, build an Authorization Header that can be handled by AuthN
	authToken, err := r.Cookie("jwt")
	if err != nil {
		util.ReturnHTTPMessage(w, r, 400, "error", "cookie not set")
		return
	}
	user, err := sp.proxyAuth(w, r, authToken.Value)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}
	sp.proxy(w, r, user)
}

func (sp ShellProxy) proxyAuth(w http.ResponseWriter, r *http.Request, token string) (*userProto.User, error) {
	r.Header.Add("Authorization", "Bearer "+token)
	user, err := rbac2.AuthenticateRequest(r, sp.authnClient)
	if err != nil {
		return &userProto.User{}, err
	}
	return user, nil
}

func (sp ShellProxy) proxy(w http.ResponseWriter, r *http.Request, user *userProto.User) {

	vars := mux.Vars(r)
	// Check if variable for vm id was passed in
	vmId := vars["vm_id"]
	if len(vmId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm id passed in")
		return
	}
	// Get the corresponding VM, if it exists
	vm, err := sp.vmClient.GetVirtualMachineById(vmId)

	if err != nil {
		glog.Errorf("did not find the right virtual machine ID")
		util.ReturnHTTPMessage(w, r, 500, "error", "no vm found")
		return
	}

	if vm.Spec.UserId != user.GetId() {
		// check if the user has access to user sessions
		impersonatedUserId := user.GetId()
		authrResponse, err := rbac2.Authorize(r, sp.authrClient, impersonatedUserId, []*authr.Permission{
			rbac2.HobbyfarmPermission(rbac2.ResourcePluralUser, rbac2.VerbGet),
			rbac2.HobbyfarmPermission(rbac2.ResourcePluralSession, rbac2.VerbGet),
			rbac2.HobbyfarmPermission(rbac2.ResourcePluralVM, rbac2.VerbGet),
		}, rbac2.OperatorAND)
		if err != nil || !authrResponse.Success {
			glog.Infof("Error doing authGrantWS %s", err)
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "access denied to connect to ssh shell session")
			return
		}
	}

	// Get the corresponding VMTemplate for the VM
	vmt, err := sp.hfClient.HobbyfarmV1().VirtualMachineTemplates(util.GetReleaseNamespace()).Get(sp.ctx, vm.Spec.VirtualMachineTemplateId, v1.GetOptions{})
	if err != nil {
		util.ReturnHTTPMessage(w, r, 404, "error", "no vm template found")
		return
	}

	// Get the target Port variable, default to 80
	targetPort := vars["port"]
	if targetPort == "" {
		targetPort = "80"
	}

	// find the corresponding service
	service := Service{}
	hasService := false
	if servicesMarhaled, ok := vmt.Spec.ConfigMap["webinterfaces"]; ok {
		servicesUnmarshaled := []Service{}
		err = json.Unmarshal([]byte(servicesMarhaled), &servicesUnmarshaled)

		if err != nil {
			glog.Infof("Error umarshaling: %v", err)
		} else {
			for _, s := range servicesUnmarshaled {
				if strconv.Itoa(s.Port) == targetPort {
					service = s
					hasService = true
					break
				}
			}
		}
	}

	// Build URL and Proxy to forward the Request to
	target := "http://127.0.0.1:" + targetPort
	remote, err := url.Parse(target)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "unable to parse URL for Localhost")
		return
	}

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

	// establish a connection to the server; retry a maximum of 5 times
	sshConn, err := retry(5, 100, func() (*ssh.Client, error) { return ssh.Dial("tcp", host+":"+port, config) })
	if err != nil {
		glog.Errorf("did not connect ssh successfully: %s", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "could not establish ssh session to vm")
		return
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(remote)
			if hasService && service.RewriteHostHeader {
				// Rewrite Host header to proxy server host (this is needed for some applications like code-server)
				r.Out.Host = r.In.Host
			}

			if hasService && service.RewriteOriginHeader {
				// Rewrite Origin header to remote host (this is needed for some applications like jupyter)
				r.Out.Header.Set("Origin", target)
			}

		},
	}
	proxy.Transport = &http.Transport{
		Dial:                sshConn.Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	//r.RequestURI = ""
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Header.Set("X-Forwarded-Proto", r.URL.Scheme)

	// Service is configured to rewrite the rootPath (default setting).
	// proxy path "/p/xyz/80/path" will be rewritten to application path "/path")
	// This is needed by applications like code-server. Some applications (like jupyter when setting a base_url) need the whole proxy path
	if !hasService || !service.NoRewriteRootPath {
		r.URL.Path = mux.Vars(r)["rest"]
	}

	proxy.ServeHTTP(w, r)
}

/*
* This is used for all connections made via the guacamole client
* Currently supported protocols are: rdp, vnc, telnet, ssh
 */
func (sp ShellProxy) ConnectGuacFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateWS(r, sp.authnClient)
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

	if vm.Spec.UserId != user.GetId() {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "you do not have access to shell")
		return
	}

	glog.Infof("Going to upgrade guac connection now... %s", vm.Name)

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

	// Resize Method has to be set in order to resize RDP Connections on the fly
	// see https://guacamole.apache.org/doc/gug/configuring-guacamole.html#rdp-display-settings
	q.Set("resize-method", "display-update")

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

type VirtualMachineInputTask struct {
	VMId              string             `json:"vm_id"`
	VMName            string             `json:"vm_name"`
	TaskInputCommands []TaskInputCommand `json:"task_command"`
}

type TaskInputCommand struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	Command             string `json:"command"`
	ExpectedOutputValue string `json:"expected_output_value"`
	ExpectedReturnCode  int    `json:"expected_return_code"`
	ReturnType 			string `json:"return_type"`
}

type VirtualMachineOutputTask struct {
	VMId               string              `json:"vm_id"`
	VMName             string              `json:"vm_name"`
	TaskOutputCommands []TaskOutputCommand `json:"task_command_output"`
}

type TaskOutputCommand struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	Command             string `json:"command"`
	ExpectedOutputValue string `json:"expected_output_value"`
	ExpectedReturnCode  int    `json:"expected_return_code"`
	ActualOutputValue   string `json:"actual_output_value"`
	ActualReturnCode    int    `json:"actual_return_code"`
	Success             bool   `json:"success"`
	ReturnType 			string `json:"return_type"`
}

func VMTaskCommandRun(task_cmd *TaskInputCommand, sess *ssh.Session) (*TaskOutputCommand, error) {
	out, err := sess.CombinedOutput(task_cmd.Command)
	actual_output_value := strings.TrimRight(string(out), "\r\n")
	actual_return_code := 0
	if err != nil {
		switch err.(type) {
		case *ssh.ExitError:
			actual_return_code = err.(*ssh.ExitError).ExitStatus()
			glog.Infof("%v", actual_return_code)
		default:
			return nil, err
		}

	}
	task_cmd_res := &TaskOutputCommand{
		Name:                task_cmd.Name,
		Description:         task_cmd.Description,
		Command:             task_cmd.Command,
		ExpectedOutputValue: task_cmd.ExpectedOutputValue,
		ExpectedReturnCode:  task_cmd.ExpectedReturnCode,
		ActualOutputValue:   actual_output_value,
		ActualReturnCode:    actual_return_code,
		Success:             task_cmd.ExpectedOutputValue == actual_output_value && task_cmd.ExpectedReturnCode == actual_return_code,
		ReturnType: 		 task_cmd.ReturnType,
	}
	return task_cmd_res, nil
}

func (sp ShellProxy) VerifyTasksFuncByVMIdGroupWithSemaphore(w http.ResponseWriter, r *http.Request) {
	// TODO: settings for define max command go routine run in same time in VM
	const MAX_COMMANDS_GO = 3
	// TODO: settings for define max try command run in VM if return code 141
	const MAX_TRY_COMMAND_RUN = 5
	user, err := rbac2.AuthenticateWS(r, sp.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
		return
	}
	// user, err := sp.authnClient.AuthN(w, r)
	// if err != nil {
	// 	util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm")
	// 	return
	// }

	var vm_input_tasks []VirtualMachineInputTask

	err = json.NewDecoder(r.Body).Decode(&vm_input_tasks)
	if err != nil {
		glog.Infof("%s", err)
	}
	glog.Infof("vm_input_tasks: %+v", vm_input_tasks)

	errorChan := make(chan error, 1)

	vm_output_tasks := make([]*VirtualMachineOutputTask, 0)
	var vm_mutex = &sync.Mutex{}
	var vm_wg sync.WaitGroup

	for _, vm_input_task := range vm_input_tasks {
		vm_wg.Add(1)

		go func(closure_vm_input_task VirtualMachineInputTask, errChan chan<- error, max_commands_go int) {
			defer vm_wg.Done()

			vmId := closure_vm_input_task.VMId
			vm, err := sp.vmClient.GetVirtualMachineById(vmId)

			if err != nil {
				glog.Errorf("did not find the right virtual machine ID")
				if len(errorChan) < cap(errorChan) {
					errChan <- err
				}
				return
			}
			if vm.Spec.UserId != user.GetId() {
				// check if the user has access to access user sessions
				// TODO: add permission like 'virtualmachine/shell' similar to 'pod/exec'
				impersonatedUserId := user.GetId()
				authrResponse, err := rbac2.Authorize(r, sp.authrClient, impersonatedUserId, []*authr.Permission{
					rbac2.HobbyfarmPermission(rbac2.ResourcePluralUser, rbac2.VerbGet),
					rbac2.HobbyfarmPermission(rbac2.ResourcePluralSession, rbac2.VerbGet),
					rbac2.HobbyfarmPermission(rbac2.ResourcePluralVM, rbac2.VerbGet),
				}, rbac2.OperatorAND)
				if err != nil || !authrResponse.Success {
					glog.Infof("Error doing authGrantWS %s", err)
					util.ReturnHTTPMessage(w, r, 403, "forbidden", "access denied to connect to ssh shell session")
					return
				}
			}
			// if vm.Spec.UserId != user.Name {
			// 	// check if the user has access to access user sessions
			// 	// TODO: add permission like 'virtualmachine/shell' similar to 'pod/exec'
			// 	_, err := sp.auth.AuthGrantWS(
			// 		rbacclient.RbacRequest().
			// 			HobbyfarmPermission("users", rbacclient.VerbGet).
			// 			HobbyfarmPermission("sessions", rbacclient.VerbGet).
			// 			HobbyfarmPermission("virtualmachines", rbacclient.VerbGet),
			// 		w, r)
			// 	if err != nil {
			// 		glog.Infof("Error doing authGrantWS %s", err)
			// 		if len(errorChan) < cap(errorChan) {
			// 			errChan <- err
			// 		}
			// 		return
			// 	}
			// }
			// ok first get the secret for the vm
			secret, err := sp.kubeClient.CoreV1().Secrets(util.GetReleaseNamespace()).Get(sp.ctx, vm.Spec.SecretName, v1.GetOptions{}) // idk?
			if err != nil {
				glog.Errorf("did not find secret for virtual machine")
				util.ReturnHTTPMessage(w, r, 500, "error", "unable to find keypair secret for vm")
				return
			}
			// // ok first get the secret for the vm
			// secret, err := sp.kubeClient.CoreV1().Secrets(util.GetReleaseNamespace()).Get(sp.ctx, vm.Spec.SecretName, v1.GetOptions{}) // idk?
			// if err != nil {
			// 	glog.Errorf("did not find secret for virtual machine")
			// 	if len(errorChan) < cap(errorChan) {
			// 		errChan <- err
			// 	}
			// 	return
			// }
			// parse the private key
			signer, err := ssh.ParsePrivateKey(secret.Data["private_key"])
			if err != nil {
				glog.Errorf("did not correctly parse private key")
				util.ReturnHTTPMessage(w, r, 500, "error", "unable to parse private key")
				return
			}
			// parse the private key
			// signer, err := ssh.ParsePrivateKey(secret.Data["private_key"])
			// if err != nil {
			// 	glog.Errorf("did not correctly parse private key")
			// 	if len(errorChan) < cap(errorChan) {
			// 		errChan <- err
			// 	}
			// 	return
			// }
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
				if len(errorChan) < cap(errorChan) {
					errChan <- err
				}
				return
			}

			commands_resp := make([]TaskOutputCommand, 0)
			var commands_mutex = &sync.Mutex{}
			var commands_wg sync.WaitGroup
			// Semaphore for count go routine run in same time in VM
			// a context is required for the weighted semaphore pkg.
			ctx := context.Background()
			var commands_semaphore = semaphore.NewWeighted(int64(max_commands_go))

			for _, task_command := range closure_vm_input_task.TaskInputCommands {
				commands_wg.Add(1)
				if err := commands_semaphore.Acquire(ctx, 1); err != nil {
					glog.Errorf("did not acquire vm_semafore")
				}
				glog.Infof("before go vm: %v, sem: %v", vmId, commands_semaphore)
				go func(closure_task_command TaskInputCommand, errChan chan<- error, max_try_command_run int) {
					defer commands_wg.Done()
					defer commands_semaphore.Release(1)
					glog.Infof("vm: %v, sem: %v", vmId, commands_semaphore)
					// try command run again when exit code == 141
					count_try_command_run := max_try_command_run
					for count_try_command_run > 0 {
						sess, err = sshConn.NewSession()
						if err != nil {
							glog.Errorf("did not setup ssh session properly")
							if len(errorChan) < cap(errorChan) {
								errChan <- err
							}
							return
						}
						if err != nil {
							glog.Infof("%s", err)
						}
						vm_task_output, err := VMTaskCommandRun(&closure_task_command, sess)

						if err != nil {
							glog.Infof("error sending command: %v", err)
							if len(errorChan) < cap(errorChan) {
								errChan <- err
							}
							return
						}
						sess.Close()
						count_try_command_run -= 1
						if vm_task_output.ActualReturnCode != 141 || count_try_command_run == 0 {
							commands_mutex.Lock()
							commands_resp = append(commands_resp, *vm_task_output)
							commands_mutex.Unlock()
							break
						}
					}
				}(task_command, errorChan, MAX_TRY_COMMAND_RUN)
			}
			commands_wg.Wait()
			vm_output_task := VirtualMachineOutputTask{
				VMId:               closure_vm_input_task.VMId,
				VMName:             closure_vm_input_task.VMName,
				TaskOutputCommands: commands_resp,
			}

			vm_mutex.Lock()
			vm_output_tasks = append(vm_output_tasks, &vm_output_task)
			vm_mutex.Unlock()
		}(vm_input_task, errorChan, MAX_COMMANDS_GO)
	}
	vm_wg.Wait()

	// Check for errors in the errorChan
	select {
	case err = <-errorChan:
		// Handle the error (log, return HTTP error response)
		close(errorChan)
		glog.Infof("Error in goroutine: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "could send command to vm")
		return
	default:
		// No error in the errorChan
		glog.Infof("No Error in goroutine: %v", vm_output_tasks)
		jsonStr, _ := json.Marshal(vm_output_tasks)
		util.ReturnHTTPContent(w, r, 200, "success", jsonStr)
	}
}

/*
* This is mainly used for SSH Connections to VMs
 */
func (sp ShellProxy) ConnectSSHFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateWS(r, sp.authnClient)
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

	if vm.Spec.UserId != user.GetId() {
		// check if the user has access to access user sessions
		// TODO: add permission like 'virtualmachine/shell' similar to 'pod/exec'
		impersonatedUserId := user.GetId()
		authrResponse, err := rbac2.Authorize(r, sp.authrClient, impersonatedUserId, []*authr.Permission{
			rbac2.HobbyfarmPermission(rbac2.ResourcePluralUser, rbac2.VerbGet),
			rbac2.HobbyfarmPermission(rbac2.ResourcePluralSession, rbac2.VerbGet),
			rbac2.HobbyfarmPermission(rbac2.ResourcePluralVM, rbac2.VerbGet),
		}, rbac2.OperatorAND)
		if err != nil || !authrResponse.Success {
			glog.Infof("Error doing authGrantWS %s", err)
			util.ReturnHTTPMessage(w, r, 403, "forbidden", "access denied to connect to ssh shell session")
			return
		}
	}

	glog.Infof("Going to upgrade connection now... %s", vm.Name)

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

func retry[T any](attempts int, sleep int, f func() (T, error)) (result T, err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			time.Sleep(time.Duration(sleep) * time.Millisecond)
			sleep *= 2
		}
		result, err = f()
		if err == nil {
			return result, nil
		}
	}
	return result, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
