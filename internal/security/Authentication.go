package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type AuthenticationProvider interface {
	AuthenticateConnection(conn net.Conn) (string, error)
	StartListener(address string) (net.Listener, error)
}

func NewMTLSAuthenticationProvider(config ServerSecurityConfig) (AuthenticationProvider, error) {
	if config.EnableMutualTLS {
		tlsConfig, err := loadTLSConfig(config)
		if err != nil {
			return nil, err
		}
		return &staticAuthN{config, tlsConfig}, nil
	}

	return &plainTextAuth{}, nil
}

type staticAuthN struct {
	ServerSecurityConfig
	tlsConfig *tls.Config
}

func (a *staticAuthN) AuthenticateConnection(conn net.Conn) (string, error) {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return "", error.New("connection was not TLS")
	}

	// Perform handshake as we may have not sent or received data yet
	// In a production server we would use a context to enforce a handshake timeout
	if !tlsConn.ConnectionState().HandshakeComplete {
		err := tlsConn.Handshake()
		if err != nil {
			return "", err // Handled by caller
		}
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return "", fmt.Errorf("no peer certificates present in incoming connection")
	}
	return strings.ToLower(state.PeerCertificates[0].Subject.CommonName), nil
}

func (a *staticAuthN) StartListener(address string) (net.Listener, error) {
	return tls.Listen(lbproxy.Protocol, address, a.tlsConfig)
}

func loadTLSConfig(config ServerSecurityConfig) (*tls.Config, error) {
	// Load CA certificate.
	caCrt := filepath.Join(config.CertPath, config.CaCertName+config.CertFileExt)
	caCert, err := os.ReadFile(caCrt)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Load Server certificate
	serverCrt := filepath.Join(config.CertPath, config.ServerCertName+config.CertFileExt)
	serverKey := filepath.Join(config.CertPath, config.ServerCertName+config.CertKeyExt)
	serverCert, err := tls.LoadX509KeyPair(serverCrt, serverKey)
	if err != nil {
		return nil, err
	}
	if err != nil {
		log.Println("Failed to load server TLS config", "ERROR:", err)
		return nil, err
	}

	// Load Client certificates
	entries, err := os.ReadDir(config.ClientsCertPath)
	if err != nil {
		return nil, err
	}
	clientCertPool := x509.NewCertPool()
	for _, e := range entries {
		if e.IsDir() {
			clientCertPath := filepath.Join(config.ClientsCertPath, e.Name(), e.Name()+config.CertFileExt)
			clientCert, err := os.ReadFile(clientCertPath)
			if err != nil {
				// In this case, we can just log and continue
				log.Println("Could not load client cert for", e.Name(), "ERROR:", err)
			} else {
				clientCertPool.AppendCertsFromPEM(clientCert)
			}
		}
	}

	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCertPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}
