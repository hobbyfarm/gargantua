package v4alpha1

// ConnectProtocol is an array of protocols that MAY be supported by the
// machine. The presence of a protocol in this list, plus an endpoint for that protocol defined
// in EnvironmentSpec will drive connection options for users.
type ConnectProtocol string

const (
	ConnectProtocolSSH  ConnectProtocol = "ssh"
	ConnectProtocolGuac ConnectProtocol = "guac"
	ConnectProtocolVNC  ConnectProtocol = "vnc"
	ConnectProtocolRDP  ConnectProtocol = "rdp"
)
