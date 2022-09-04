package security

import (
	"crypto/tls"
	"fmt"
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"log"
	"net"
	"strings"
)

type AuthenticationProvider interface {
	AuthenticateConnection(conn net.Conn) (string, error)
	StartListener(address string) (net.Listener, error)
}

type StaticAuthN struct {
	CertFilePath string
	KeyFilePath  string
	CACommonName string
}

func (a *StaticAuthN) AuthenticateConnection(conn net.Conn) (string, error) {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return "", fmt.Errorf("connetion was not TLS")
	}

	// Perform handshake as we may have not sent or received data yet
	// In a production server we would use a context to enforce a handshake timeout
	err := tlsConn.Handshake()
	if err != nil {
		return "", err
	}

	state := tlsConn.ConnectionState()
	leafCert := state.PeerCertificates[0]
	if strings.EqualFold(leafCert.Issuer.CommonName, a.CACommonName) {
		return strings.ToLower(leafCert.Subject.CommonName), nil
	}

	return "", fmt.Errorf("could not client certificate under CA CN=" + a.CACommonName)
}

func (a *StaticAuthN) StartListener(address string) (net.Listener, error) {
	cert, err := tls.LoadX509KeyPair(a.CertFilePath, a.KeyFilePath)
	if err != nil {
		return nil, err
	}
	if err != nil {
		log.Println("Failed to load TLS config", "ERROR:", err)
		return nil, err
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	return tls.Listen(lbproxy.Protocol, address, config)
}
