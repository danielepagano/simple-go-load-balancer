package main

import (
	"github.com/danielepagano/teleport-int-load-balancer/internal"
	"log"
	"os"
	"os/signal"
)

func main() {
	log.Println("Initializing Load-balancing Proxy")
	// This would be the place to load config from params, env vars etc. without changing anything else
	config := internal.GetStaticConfig()

	for _, app := range config.Apps {
		s := &internal.ProxyServer{
			App:                    app,
			DefaultRateLimitConfig: config.DefaultRateLimitConfig,
		}
		appId := app.AppId // fix value from loop

		// Async start each app; server will not panic if some apps fail to start (usually port busy)
		// This would be a pretty loud alert in a real system
		go func() {
			err := s.StartServer()
			if err != nil {
				log.Println("ERROR - Application failed to start:", appId, "ERROR:", err)
			}
		}()
	}

	// Wait until Ctrl-C or equivalent
	sigInt := make(chan os.Signal, 1)
	signal.Notify(sigInt, os.Interrupt)
	<-sigInt

	log.Println("bye.")
	os.Exit(0)
}
