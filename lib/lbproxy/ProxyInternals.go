package lbproxy

import (
	"io"
	"log"
	"math"
	"net"
	"strings"
	"sync"
)

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

func (a *application) SubmitConnection(client net.Conn, rlm RateLimitManager, stopSignal chan bool) {
	// Rate-limit exceeded; close client connection
	appId := a.config.Name
	if !rlm.AddConnection() {
		log.Println("Rate limit exceeded for app", appId)
		a.closeClientConnection(client, rlm)
	} else {
		defer a.closeClientConnection(client, rlm)

		for {
			select {
			case <-stopSignal:
				return
			default:
				a.proxyConnection(client, rlm)
				// Stop and close client connection
				return
			}
		}
	}
}

func (a *application) proxyConnection(clientConn net.Conn, rlm RateLimitManager) {
	upStream := a.acquireUpstream()
	// defer a.releaseUpstream(upStream)
	upstreamConn, err := net.DialTimeout("tcp", upStream, UpstreamTimeout)

	if err != nil {
		log.Println("Error connecting to upstream", upStream, "ERR:", err)
		// Give up. In a more mature system, we'd quarantine this upstream and try another upstream
		a.releaseUpstream(upStream)
		a.closeClientConnection(clientConn, rlm)
		return
	}
	// defer a.closeUpstreamConnection(upstreamConn, upStream)
	serverClosed := make(chan bool, 1)
	clientClosed := make(chan bool, 1)

	// TODO: not quite right... double-closes and no client close on server kill

	go a.sendToUpstream(clientConn, upstreamConn, rlm, clientClosed)
	go a.returnToClient(upstreamConn, clientConn, upStream, serverClosed)

	var fullyClosed chan bool
	select {
	case <-clientClosed:
		a.closeUpstreamConnection(upstreamConn, upStream)
		fullyClosed = serverClosed
	case <-serverClosed:
		a.closeClientConnection(clientConn, rlm)
		fullyClosed = clientClosed
	}

	// Wait here, so that we wait to perform deferred close actions
	<-fullyClosed
}

func (a *application) sendToUpstream(upstreamConn, clientConn net.Conn, rlm RateLimitManager, srcClosed chan bool) {
	_, err := io.Copy(upstreamConn, clientConn)

	if err != nil {
		log.Println("Copy error", err)
	}

	a.closeClientConnection(clientConn, rlm)
	srcClosed <- true
}

func (a *application) returnToClient(clientConn, upstreamConn net.Conn, upStream string, srcClosed chan bool) {
	_, err := io.Copy(clientConn, upstreamConn)

	if err != nil {
		log.Println("Copy error", err)
	}

	a.closeUpstreamConnection(upstreamConn, upStream)
	srcClosed <- true
}

func (a *application) closeClientConnection(c net.Conn, rlm RateLimitManager) {
	err := c.Close()
	alreadyClosed := err != nil && strings.Contains(err.Error(), "use of closed network connection")
	if err != nil {
		log.Println("Failed to close connection to", c.RemoteAddr(), "ERROR", err)
	}

	// If the error was that the connection was already closed, do not perform connection accounting done
	if err == nil && !alreadyClosed {
		rlm.ReleaseConnection()
	}
}

func (a *application) closeUpstreamConnection(c net.Conn, upstream string) {
	err := c.Close()
	alreadyClosed := err != nil && strings.Contains(err.Error(), "use of closed network connection")
	if err != nil && !alreadyClosed {
		log.Println("Failed to close connection to", c.RemoteAddr(), "ERROR", err)
	}

	// If the error was that the connection was already closed, do not perform connection accounting done
	if err == nil && !alreadyClosed {
		a.releaseUpstream(upstream)
	}
}

func (a *application) acquireUpstream() string {
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

func (a *application) releaseUpstream(upstream string) string {
	a.routingLock.Lock()
	defer a.routingLock.Unlock()
	if a.upstreamConn[upstream] > 0 {
		a.upstreamConn[upstream] -= 1
	}
	log.Println("Released upstream", upstream, "LOAD:", a.upstreamConn)
	return upstream
}
