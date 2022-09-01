package lbproxy

import (
	"io"
	"log"
	"math"
	"net"
	"sync"
)

const LogClosedConnErrors = false

func InitApplication(config ApplicationConfig) Application {
	app := &application{
		config:       config,
		routingLock:  sync.RWMutex{},
		upstreamConn: map[string]int{},
	}
	for _, u := range config.Upstreams {
		app.upstreamConn[u.Address] = 0
	}
	return app
}

type application struct {
	config       ApplicationConfig
	routingLock  sync.RWMutex
	upstreamConn map[string]int
}

func (a *application) SubmitConnection(client net.Conn, rlm RateLimitManager) {
	// Rate-limit exceeded; close client connection
	appId := a.config.Name
	if !rlm.AddConnection() {
		log.Println("Rate limit exceeded for app", appId)
		// Close connection; it was never added to count of open connections in rlm;
		// No further clean-up necessary
		a.closeConnection(client)
	} else {
		// Release the connection from RLM after proxying is completed
		defer rlm.ReleaseConnection()
		a.proxyConnection(client)
	}
}

func (a *application) proxyConnection(clientConn net.Conn) {
	// Use an upstream connection within this scope
	upStream := a.acquireUpstream()
	defer a.releaseUpstream(upStream)

	// Close client connection when completed or denied
	defer a.closeConnection(clientConn)

	// Negotiate an upstream connection
	tcpAddress, err := net.ResolveTCPAddr(Protocol, upStream)
	if err != nil {
		log.Println(a.config.Name, ": could resolve upstream address", upStream, "ERROR:", err)
		// Configuration issue; in a more mature system, we would remove this upstream from the list and raise an alert
		return
	}
	upstreamConn, err := net.DialTCP("tcp", nil, tcpAddress)
	if err != nil {
		log.Println(a.config.Name, ": error connecting to upstream", upStream, "ERR:", err)
		// Give up and disconnect client.
		// In a more mature system, we'd quarantine this upstream and try another upstream
		return
	}

	defer a.closeConnection(upstreamConn)

	aSourceClosed := make(chan struct{}, 1)

	go a.pipe(clientConn, upstreamConn, aSourceClosed)
	go a.pipe(upstreamConn, clientConn, aSourceClosed)

	// Wait until one side sends EOF or has error, at which point we'll exit this,
	// which will hit the deferred closes and wrap up everything
	<-aSourceClosed

	// Note that we will routinely attempt to close some connection after they are already closed
	// We could avoid this with some more complex coordination, at the risk of more concurrency issues
}

func (a *application) pipe(dest, source net.Conn, srcClosed chan struct{}) {
	// If we wanted to implement bandwidth rate-limiting/throttling, we would need to
	// manually copy the data between the connections, as io.Copy continues until error or EOF
	_, err := io.Copy(dest, source)

	if err != nil && (LogClosedConnErrors || !IsErrorClosedNetworkConnection(err)) {
		log.Println("Network IO error", err)
	}

	srcClosed <- struct{}{}
}

func (a *application) closeConnection(c net.Conn) {
	err := c.Close()
	if err != nil && (LogClosedConnErrors || !IsErrorClosedNetworkConnection(err)) {
		log.Println("Failed to close connection to", c.RemoteAddr(), "ERROR", err)
	}
}

func (a *application) acquireUpstream() string {
	// Thread-safe map operation to find and increase active connections per upstream
	// Follow with defer acquireUpstream()
	a.routingLock.Lock()
	defer a.routingLock.Unlock()
	minConn := math.MaxInt
	var upstream string
	for k, v := range a.upstreamConn {
		if v < minConn {
			upstream = k
			minConn = v
		}
	}
	a.upstreamConn[upstream] += 1
	log.Println("Acquired upstream", upstream, "LOAD:", a.upstreamConn)
	return upstream
}

func (a *application) releaseUpstream(upstream string) {
	// Tracks a released connection from an upstream
	a.routingLock.Lock()
	defer a.routingLock.Unlock()
	if a.upstreamConn[upstream] > 0 {
		a.upstreamConn[upstream] -= 1
	}
	log.Println("Released upstream", upstream, "LOAD:", a.upstreamConn)
	return
}
