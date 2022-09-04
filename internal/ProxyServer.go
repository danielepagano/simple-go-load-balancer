package internal

import (
	"errors"
	"fmt"
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"log"
	"net"
)

const localServerPrefix = ":"

type ProxyServerConfig struct {
	App             AppConfig
	RateLimitConfig lbproxy.RateLimitManagerConfig
	Authn           AuthenticationProvider
	Authz           AuthorizationProvider
}

type ProxyServer struct {
	config       ProxyServerConfig
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
		config:       config,
		rateManagers: make(map[string]lbproxy.RateLimitManager),
	}, nil
}

func (s *ProxyServer) Start() error {
	listener, err := s.startListener()
	if err != nil {
		return err
	}
	defer s.closeListener(listener)

	// Creates the application that will proxy and load-balance the incoming traffic
	app := s.config.App
	lbProxyApp := lbproxy.InitApplication(app.ToApplicationConfig())
	log.Println("STARTED APP", app.AppId, "on port", app.ProxyPort)

	// Listen loop
	// Currently we accept connections from a single thread per app,
	// so no need worry about concurrent access to the rateManagers map
	// A possible optimization could be to perform some of this work in a goroutine
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println("APP", app.AppId, "Failed to accept client connection", conn, "ERROR:", err)
			// If connection was closed, it means listener was closed; exit the application server
			// otherwise, we try and can accept future connections
			if errors.Is(err, net.ErrClosed) {
				return err
			}
		} else {
			clientId, authorized := s.ensureSecured(conn)
			if authorized {
				s.handoffConnection(clientId, lbProxyApp, conn)
			} else {
				err := conn.Close()
				if err != nil {
					log.Println("APP", app.AppId, "Failed to close denied client connection", conn, "ERROR", err)
				}
			}
		}
	}
}

func (s *ProxyServer) ensureSecured(conn *net.TCPConn) (string, bool) {
	app := s.config.App
	clientId, err := s.config.Authn.AuthenticateConnection(conn)
	if err != nil {
		log.Println("APP", app.AppId, "Failed to authenticate client connection", conn, "ERROR:", err)
		return clientId, false
	}

	authorized, err := s.config.Authz.AuthorizeClient(clientId, app.AppId)
	if err != nil {
		log.Println("APP", app.AppId, "Failed to authorize clientId", clientId, "ERROR:", err)
		return clientId, false
	}

	// Extra logging for clarity
	if !authorized {
		log.Println("APP", app.AppId, "Access denied to clientId", clientId)
	}
	return clientId, authorized

}

func (s *ProxyServer) handoffConnection(clientId string, lbProxyApp lbproxy.Application, conn *net.TCPConn) {
	// Creates one rate-limit manager per (app,clientId)
	// If we wanted to track rate limits across app for each client, we would create a thread-safe
	// wrapper around a map that would be injected by the caller, so method app would do something like
	// rlmStore.getRateLimitManager(clientId), which would get or create the instance for this client
	var rlm lbproxy.RateLimitManager
	var found bool
	if rlm, found = s.rateManagers[clientId]; !found {
		rlm = lbproxy.CreateRateLimitManager(clientId+"@"+s.config.App.AppId, s.config.RateLimitConfig)
		s.rateManagers[clientId] = rlm
	}

	// Hand off the connection to lbproxy and prepare to receive a new one
	go lbProxyApp.SubmitConnection(conn, rlm)
}

func (s *ProxyServer) startListener() (*net.TCPListener, error) {
	tcpAddress, err := net.ResolveTCPAddr(lbproxy.Protocol, localServerPrefix+s.config.App.ProxyPort)
	if err != nil {
		return nil, fmt.Errorf("could resolve local TCP address for listening on port "+s.config.App.ProxyPort, err)
	}

	listener, err := net.ListenTCP(lbproxy.Protocol, tcpAddress)
	if err != nil {
		log.Println("Failed to listen for tcp connections on", tcpAddress, "ERROR:", err)
		return nil, err
	}
	return listener, nil
}

func (s *ProxyServer) closeListener(ln *net.TCPListener) {
	err := ln.Close()
	if err != nil {
		log.Println("Failed to close listener for app", s.config.App.AppId, "ERROR:", err)
	}
}
