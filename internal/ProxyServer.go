package internal

import (
	"errors"
	"fmt"
	"github.com/danielepagano/teleport-int-load-balancer/internal/security"
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"log"
	"net"
)

const localServerPrefix = ":"

type ProxyServerConfig struct {
	App             AppConfig
	RateLimitConfig lbproxy.RateLimitManagerConfig
	Authn           security.AuthenticationProvider
	Authz           security.AuthorizationProvider
}

type ProxyServer struct {
	ProxyServerConfig
	rateManagers map[string]lbproxy.RateLimitManager
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
				s.handoffConnection(clientId, lbProxyApp, conn)
			}
		}
	}
}

func (s *ProxyServer) ensureSecured(conn net.Conn) (string, error) {
	app := s.App
	clientId, err := s.Authn.AuthenticateConnection(conn)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate client connection from %v. %w", conn.RemoteAddr(), err)
	}

	authorized, err := s.Authz.AuthorizeClient(clientId, app.AppId)
	if err != nil {
		return "", fmt.Errorf("failed to authorize clientId %v. %w", clientId, err)
	}

	// Extra logging for clarity
	if !authorized {
		return "", fmt.Errorf("app access denied clientId %v", clientId)
	}
	return clientId, nil
}

func (s *ProxyServer) handoffConnection(clientId string, lbProxyApp lbproxy.Application, conn net.Conn) {
	// Creates one rate-limit manager per (app,clientId)
	// If we wanted to track rate limits across app for each client, we would create a thread-safe
	// wrapper around a map that would be injected by the caller, so method app would do something like
	// rlmStore.getRateLimitManager(clientId), which would get or create the instance for this client
	var rlm lbproxy.RateLimitManager
	var found bool
	if rlm, found = s.rateManagers[clientId]; !found {
		rlm = lbproxy.CreateRateLimitManager(clientId+"@"+s.App.AppId, s.RateLimitConfig)
		s.rateManagers[clientId] = rlm
	}

	// Hand off the connection to lbproxy and prepare to receive a new one
	go lbProxyApp.SubmitConnection(conn, rlm)
}

func (s *ProxyServer) startListener() (net.Listener, error) {
	address := localServerPrefix + s.App.ProxyPort
	listener, err := s.Authn.StartListener(address)
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
