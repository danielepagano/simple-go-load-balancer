package main

import (
	"github.com/danielepagano/teleport-int-load-balancer/internal"
	"log"
	"os"
	"os/signal"
)

func main() {
	config := internal.GetStaticConfig()
	stopSignal := make(chan bool, 1)

	for _, app := range config.Apps {
		s := &internal.ProxyServer{
			App:                    app,
			DefaultRateLimitConfig: config.DefaultRateLimitConfig,
		}
		appId := app.AppId
		go func() {
			log.Println("Starting " + appId)
			err := s.StartServer(stopSignal)
			if err != nil {
				log.Println("Application failed to start:", appId)
			}
		}()
	}

	// Wait until Ctrl-C or equivalent
	sigInt := make(chan os.Signal, 1)
	signal.Notify(sigInt, os.Interrupt)
	<-sigInt

	// Signal apps to stop
	stopSignal <- true
	log.Println("bye.")
}
