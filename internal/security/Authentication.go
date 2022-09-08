package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type Authenticator interface {
	GetCurrentTlsConfig() *tls.Config
	AuthenticateConnection(conn net.Conn) (string, error)
}

func NewAuthenticator(config ServerSecurityConfig) (Authenticator, error) {
	tlsConfig, err := loadTLSConfig(config)
	if err != nil {
		return nil, err
	}
	return &staticAuthN{config, tlsConfig}, nil
}

type staticAuthN struct {
	ServerSecurityConfig
	tlsConfig *tls.Config
}

func (a *staticAuthN) AuthenticateConnection(conn net.Conn) (string, error) {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return "", fmt.Errorf("connection was not TLS")
	}

	// Perform handshake as we may have not sent or received data yet
	// In a production server we would use a context to enforce a handshake timeout
	err := tlsConn.Handshake()
	if err != nil {
		return "", err // Handled by caller
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return "", fmt.Errorf("no peer certificates present in incoming connection")
	}
	return strings.ToLower(state.PeerCertificates[0].Subject.CommonName), nil
}

func (a *staticAuthN) GetCurrentTlsConfig() *tls.Config {
	return a.tlsConfig
}

func loadTLSConfig(config ServerSecurityConfig) (*tls.Config, error) {
	// Load CA certificate.
	caCrt := filepath.Join(config.CaCert)
	caCert, err := os.ReadFile(caCrt)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("could not append CA certificate to pool")
	}

	// Load Server certificate
	serverCrt := filepath.Join(config.ServerCert)
	serverKey := filepath.Join(config.ServerKey)
	serverCert, err := tls.LoadX509KeyPair(serverCrt, serverKey)
	if err != nil {
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
			clientCertPath := filepath.Join(config.ClientsCertPath, e.Name(), e.Name()+config.ClientCertFileExt)
			clientCert, err := os.ReadFile(clientCertPath)
			if err != nil {
				// In this case, we can just log and continue
				log.Println("Could not load client cert for", e.Name(), "ERROR:", err)
			} else {
				if ok := clientCertPool.AppendCertsFromPEM(clientCert); !ok {
					log.Println("Could not append client cert for", e.Name(), "to pool.", "ERROR:", err)
				}
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
