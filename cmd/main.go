package main

import (
	"github.com/danielepagano/teleport-int-load-balancer/internal"
	"github.com/danielepagano/teleport-int-load-balancer/internal/security"
	"github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"
	"log"
	"os"
	"os/signal"
)

func main() {
	log.Println("Initializing Load-balancing Proxy")
	// This would be the place to load config from params, env vars etc. without changing anything else
	config := internal.GetStaticConfig()

	var authn security.AuthenticationProvider
	var authz security.AuthorizationProvider

	if config.SecurityConfig.EnableMutualTLS {
		authn = &security.StaticAuthN{
			CertFilePath: config.SecurityConfig.CertFilePath,
			KeyFilePath:  config.SecurityConfig.KeyFilePath,
		}
		authz = &security.SimpleAuthZ{
			ClientPermissions: config.Clients,
		}
	} else {
		authn = &security.PlainTextAuth{}
		authz = &security.NoOpAuthZ{}
	}

	for _, app := range config.Apps {
		// Async start each app; server will not panic if some apps fail to start (usually port busy)
		// This would be a pretty loud alert in a real system
		go startAppServer(app, config.DefaultRateLimitConfig, authn, authz)
	}

	// Wait until Ctrl-C or equivalent
	sigInt := make(chan os.Signal, 1)
	signal.Notify(sigInt, os.Interrupt)
	<-sigInt

	log.Println("bye.")
}

func startAppServer(app internal.AppConfig, rateLimitConfig lbproxy.RateLimitManagerConfig,
	authn security.AuthenticationProvider, authz security.AuthorizationProvider) {
	serverConfig := internal.ProxyServerConfig{
		App:             app,
		RateLimitConfig: rateLimitConfig,
		Authn:           authn,
		Authz:           authz,
	}

	// Initialize and check configuration
	server, err := internal.NewProxyServer(serverConfig)
	if err != nil {
		log.Println("ERROR - could not initialise server for", app.AppId, "ERROR:", err)
		return
	}

	// Try and start server
	err = server.Start()
	if err != nil {
		log.Println("ERROR - server failed to start for", app.AppId, "ERROR:", err)
	}
}
