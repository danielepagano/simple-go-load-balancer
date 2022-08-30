package internal

import (
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"log"
	"net"
)

const PROTOCOL = "tcp"

type ProxyServer struct {
	App                    AppConfig
	DefaultRateLimitConfig lbproxy.RateLimitManagerConfig
}

func (s *ProxyServer) StartServer(stopSignal chan bool) error {
	tcpAddress, _ := net.ResolveTCPAddr(PROTOCOL, ":"+s.App.ProxyPort)
	ln, err := net.ListenTCP(PROTOCOL, tcpAddress)
	if err != nil {
		log.Println("Failed to listen for tcp connections on", tcpAddress, "ERROR:", err)
		return err
	}
	defer func(ln *net.TCPListener) {
		err := ln.Close()
		if err != nil {
			log.Println("Failed to close listener on", tcpAddress, "ERROR:", err)
		}
	}(ln)

	// Creates the application that will proxy and load-balance the incoming traffic
	app := lbproxy.InitApplication(s.App.ToApplicationConfig())

	// Creates a rate-limit scope at the application level
	// TODO: move this lower to client level once we have auth
	rlm := lbproxy.CreateRateLimitManager(s.DefaultRateLimitConfig)

	log.Println(s.App.AppId, "started and listening on", tcpAddress)

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Println("Failed to accept connection", conn, "ERROR:", err)
			continue
		}
		log.Println("Client", conn.RemoteAddr(), "connected")
		// TODO: perform auth to get client id and validate access
		go app.SubmitConnection(conn, rlm, stopSignal)
	}
}
