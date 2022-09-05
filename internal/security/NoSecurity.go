package security

import (
	"fmt"
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"net"
)

type plainTextAuth struct {
}

func (a *plainTextAuth) AuthenticateConnection(conn net.Conn) (string, error) {
	return "localhost", nil
}

func (a *plainTextAuth) StartListener(address string) (net.Listener, error) {
	tcpAddress, err := net.ResolveTCPAddr(lbproxy.Protocol, address)
	if err != nil {
		return nil, fmt.Errorf("could resolve local TCP address for listening on "+address, err)
	}
	return net.ListenTCP(lbproxy.Protocol, tcpAddress)
}

type noOpAuthZ struct {
}

func (a *noOpAuthZ) AuthorizeClient(clientId string, appId string) (bool, error) {
	return true, nil
}
