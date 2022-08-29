package lbproxy

import (
	"net"
	"time"
)

const UpstreamTimeout = 5 * time.Second

// Application represents a provisioned group of upstream servers that are being load-balanced.
// Typically, a server can expose one or more applications, one per open TCP port
type Application interface {
	// SubmitConnection hands off a client connection to load-balance it against one of the upstream servers
	// After the connection is submitted, the Application instance will decide whether it will be connected
	// or not, and otherwise close it and manage any errors.
	// stopSignal can be signaled at any time by the caller to interrupt the connection, e.g. if server is shutting down
	SubmitConnection(clientConnection net.Conn, rateLimitManager RateLimitManager, stopSignal chan bool)
}

// ApplicationConfig initializes an Application instance
type ApplicationConfig struct {
	Name      string           // Used for diagnostic logging
	Upstreams []UpstreamServer // Upstream servers to use
}

// UpstreamServer describes a server being load-balanced
type UpstreamServer struct {
	Address string // Server address as would be accepted by a TCP Dial
}
