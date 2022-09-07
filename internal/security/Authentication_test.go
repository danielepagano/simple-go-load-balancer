package security

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Warning: these certs will expire after Sept 2022
// In non-sample app, we would be generating these certs directly in the test
var config = ServerSecurityConfig{
	ClientsCertPath:   "../../certs/clients",
	ClientCertFileExt: ".crt",
	ClientCertKeyExt:  ".key",
	CaCert:            "../../certs/ca.crt",
	ServerCert:        "../../certs/server.crt",
	ServerKey:         "../../certs/server.key",
}

const clientId = "localhost"
const serverAddress = "localhost:9001"

func TestMTLSAuthenticationProvider(t *testing.T) {
	auth, tlsListener := startTLSListener(t)
	t.Cleanup(func() {
		_ = tlsListener.Close()
	})

	clientConfig := loadClientTLSConfig(t, clientId)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		clientConn, acceptErr := tlsListener.Accept()
		defer clientConn.Close()
		defer wg.Done()

		t.Log("TLS listener accepted incoming")
		if acceptErr != nil {
			t.Errorf("Could not accept incoming error = %v", acceptErr)
		}
		authClientId, acceptErr := auth.AuthenticateConnection(clientConn)
		if acceptErr != nil {
			t.Errorf("Could not authenticate incoming error = %v", acceptErr)
			return
		}
		if authClientId != clientId {
			t.Errorf("Inccorecct clientId: %v", authClientId)
			return
		}
		t.Log("SUCCESS: authenticated clientId " + authClientId)
	}()

	conn, err := tls.Dial("tcp", serverAddress, clientConfig)
	t.Cleanup(func() {
		_ = conn.Close()
	})
	if err != nil {
		t.Fatalf("could not connect to server err = %v", err)
	}

	_, err = io.WriteString(conn, "Hello mTLS\n")
	if err != nil {
		t.Fatalf("could not write to server = %v", err)
	}

	wg.Wait()
}

func loadClientTLSConfig(t *testing.T, clientId string) *tls.Config {
	t.Helper()
	// Load CA certificate.
	caCrt := filepath.Join(config.CaCert)
	caCert, err := os.ReadFile(caCrt)
	if err != nil {
		t.Fatalf("could not load CA cert = %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Load Client certificate
	clientCrt := filepath.Join(config.ClientsCertPath, clientId, clientId+config.ClientCertFileExt)
	clientKey := filepath.Join(config.ClientsCertPath, clientId, clientId+config.ClientCertKeyExt)
	clientCert, err := tls.LoadX509KeyPair(clientCrt, clientKey)
	if err != nil {
		t.Fatalf("could not load client cert = %v", err)
	}

	return &tls.Config{
		RootCAs: caCertPool,
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &clientCert, nil
		},
	}
}

func startTLSListener(t *testing.T) (Authenticator, net.Listener) {
	t.Helper()
	auth, err := NewAuthenticator(config)
	if err != nil {
		t.Fatalf("TestMTLSAuthenticationProvider() error = %v", err)
	}

	tlsConfig := auth.GetCurrentTlsConfig()
	tlsListener, err := tls.Listen(lbproxy.Protocol, serverAddress, tlsConfig)
	if err != nil {
		t.Fatalf("TestMTLSAuthenticationProvider() error = %v", err)
	}
	t.Log("TLS Listener started on", tlsListener.Addr().String())
	return auth, tlsListener
}
