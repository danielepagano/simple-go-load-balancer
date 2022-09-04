package security

import (
	"fmt"
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"net"
)

type PlainTextAuth struct {
}

func (a *PlainTextAuth) AuthenticateConnection(conn net.Conn) (string, error) {
	return "localhost", nil
}

func (a *PlainTextAuth) StartListener(address string) (net.Listener, error) {
	tcpAddress, err := net.ResolveTCPAddr(lbproxy.Protocol, address)
	if err != nil {
		return nil, fmt.Errorf("could resolve local TCP address for listening on "+address, err)
	}
	return net.ListenTCP(lbproxy.Protocol, tcpAddress)
}

type NoOpAuthZ struct {
}

func (a *NoOpAuthZ) AuthorizeClient(clientId string, appId string) (bool, error) {
	return true, nil
}
