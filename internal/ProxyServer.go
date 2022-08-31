package internal

import (
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"log"
	"net"
)

const LocalServerPrefix = ":"

type ProxyServer struct {
	App                    AppConfig
	DefaultRateLimitConfig lbproxy.RateLimitManagerConfig
	rateManagers           map[string]lbproxy.RateLimitManager
}

func (s *ProxyServer) StartServer() error {
	s.rateManagers = make(map[string]lbproxy.RateLimitManager)

	listener, err := s.startListener()
	if err != nil {
		return err
	}
	defer s.closeListener(listener)

	// Creates the application that will proxy and load-balance the incoming traffic
	app := lbproxy.InitApplication(s.App.ToApplicationConfig())
	log.Println("STARTED APP", s.App.AppId, "on port", s.App.ProxyPort)

	// Listen loop
	// Currently we accept connections from a single thread per app,
	// so no need worry about concurrent access to the rateManagers map
	// A possible optimization could be to perform some of this work in a goroutine
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println("APP", s.App.AppId, "Failed to accept client connection", conn, "ERROR:", err)
			continue
		}

		// TODO: temporarily until auth, the clientId is just the remote address
		clientId := "localhost"

		// Creates one rate-limit manager per (app,clientId)
		// If we wanted to track rate limits across app for each client, we would create a thread-safe
		// wrapper around a map that would be injected by the caller, so method app would do something like
		// rlmStore.getRateLimitManager(clientId), which would get or create the instance for this client
		var rlm lbproxy.RateLimitManager
		var found bool
		if rlm, found = s.rateManagers[clientId]; !found {
			rlm = lbproxy.CreateRateLimitManager(clientId+"@"+s.App.AppId, s.DefaultRateLimitConfig)
			s.rateManagers[clientId] = rlm
		}

		// Hand off the connection to lbproxy and prepare to receive a new one
		go app.SubmitConnection(conn, rlm)
	}
}

func (s *ProxyServer) startListener() (*net.TCPListener, error) {
	tcpAddress, err := net.ResolveTCPAddr(lbproxy.Protocol, LocalServerPrefix+s.App.ProxyPort)
	if err != nil {
		log.Println("Could resolve local TCP address for listening on port", s.App.ProxyPort, "ERROR:", err)
		return nil, err
	}

	listener, err := net.ListenTCP(lbproxy.Protocol, tcpAddress)
	if err != nil {
		log.Println("Failed to listen for tcp connections on", tcpAddress, "ERROR:", err)
		return nil, err
	}
	return listener, nil
}

func (s *ProxyServer) closeListener(listener *net.TCPListener) {
	func(ln *net.TCPListener) {
		err := ln.Close()
		if err != nil {
			log.Println("Failed to close listener for app", s.App.AppId, "ERROR:", err)
		}
	}(listener)
}
