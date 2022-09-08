package internal

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/danielepagano/teleport-int-load-balancer/internal/security"
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"log"
	"net"
	"sync"
)

const localServerPrefix = ":"

type ProxyServerConfig struct {
	App             AppConfig
	RateLimitConfig lbproxy.RateLimitManagerConfig
	Authn           security.Authenticator
	Authz           security.Authorizer
}

type ProxyServer struct {
	ProxyServerConfig
	rateManagersLock sync.RWMutex
	rateManagers     map[string]lbproxy.RateLimitManager
}

func NewProxyServer(config ProxyServerConfig) (*ProxyServer, error) {
	if config.RateLimitConfig.MaxOpenConnections == 0 ||
		config.RateLimitConfig.MaxRateAmount == 0 {
		return nil, fmt.Errorf("application has zero allowed rate")
	}
	if len(config.App.Upstreams) == 0 {
		return nil, fmt.Errorf("at least one upstream per app is required")
	}

	return &ProxyServer{
		ProxyServerConfig: config,
		rateManagers:      make(map[string]lbproxy.RateLimitManager),
	}, nil
}

func (s *ProxyServer) Start() error {
	listener, err := s.startListener()
	if err != nil {
		return err
	}
	defer s.closeListener(listener)

	// Creates the application that will proxy and load-balance the incoming traffic
	lbProxyApp := lbproxy.InitApplication(s.App.ToApplicationConfig())
	log.Println("STARTED APP", s.App.AppId, "on port", s.App.ProxyPort)

	// Listen loop
	// Currently we accept connections from a single thread per app,
	// so no need worry about concurrent access to the rateManagers map
	// A possible optimization could be to perform some of this work in a goroutine
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("APP", s.App.AppId, "Failed to accept client connection from", conn.RemoteAddr(), "ERROR:", err)
			// If connection was closed, it means listener was closed; exit the application server
			// otherwise, we try and can accept future connections
			if errors.Is(err, net.ErrClosed) {
				return err
			}
		} else {
			go s.authorizeAndHandoffConnection(lbProxyApp, conn)
		}
	}
}

func (s *ProxyServer) authorizeAndHandoffConnection(lbProxyApp lbproxy.Application, conn net.Conn) {
	clientId, err := s.ensureSecured(conn)
	if err != nil {
		if err != nil {
			log.Println("APP", s.App.AppId, "Could not authorize client connection", "ERROR", err)
		}
		err := conn.Close()
		if err != nil {
			log.Println("APP", s.App.AppId, "Failed to close denied client connection from", conn.RemoteAddr(), "ERROR", err)
		}
	} else {
		rlm := s.getRateLimitManager(clientId)
		go lbProxyApp.SubmitConnection(conn, rlm)
	}
}

func (s *ProxyServer) ensureSecured(conn net.Conn) (string, error) {
	app := s.App
	clientId, err := s.Authn.AuthenticateConnection(conn)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate client connection from %v. %w", conn.RemoteAddr(), err)
	}

	err = s.Authz.AuthorizeClient(clientId, app.AppId)
	return clientId, err
}

func (s *ProxyServer) getRateLimitManager(clientId string) lbproxy.RateLimitManager {
	// Creates one rate-limit manager per (app,clientId)
	s.rateManagersLock.Lock()
	defer s.rateManagersLock.Unlock()
	var rlm lbproxy.RateLimitManager
	var found bool
	if rlm, found = s.rateManagers[clientId]; !found {
		rlm = lbproxy.CreateRateLimitManager(clientId+"@"+s.App.AppId, s.RateLimitConfig)
		s.rateManagers[clientId] = rlm
	}
	return rlm
}

func (s *ProxyServer) startListener() (net.Listener, error) {
	address := localServerPrefix + s.App.ProxyPort
	tlsConfig := s.Authn.GetCurrentTlsConfig()
	listener, err := tls.Listen(lbproxy.Protocol, address, tlsConfig)
	if err != nil {
		log.Println("APP", s.App.AppId, "Failed to listen for tcp connections on", address, "ERROR:", err)
		return nil, err
	}
	return listener, nil
}

func (s *ProxyServer) closeListener(ln net.Listener) {
	err := ln.Close()
	if err != nil {
		log.Println("APP", s.App.AppId, "Failed to close listener", "ERROR:", err)
	}
}
