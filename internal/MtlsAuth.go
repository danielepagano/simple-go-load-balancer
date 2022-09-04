package internal

import "net"

type AuthenticationProvider interface {
	AuthenticateConnection(conn net.Conn) (string, error)
}

type StaticAuthN struct {
}

func (a *StaticAuthN) AuthenticateConnection(conn net.Conn) (string, error) {
	return "localhost", nil
}
