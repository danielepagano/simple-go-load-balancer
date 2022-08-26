package lbproxy

import (
	"net"
)

// Application represents a provisioned group of upstream servers that are being load-balanced.
// Typically, a server can expose one or more applications, one per open TCP port
type Application interface {
	submitConnection(clientConnection net.TCPConn, rateLimitManager RateLimitManager)
}

// TODO: Create Application instance
// func InitApplication(config ApplicationConfig) Application {}

// TODO: implement interface

// ApplicationConfig initializes an Application instance
type ApplicationConfig struct {
	name      string
	upstreams []UpstreamServer
}

// UpstreamServer describes a server being load-balanced
type UpstreamServer struct {
	address string // Server address as would be accepted by a TCP Dial
}
