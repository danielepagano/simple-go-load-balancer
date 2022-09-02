package lbproxy

import (
	"log"
	"net"
	"sync"
	"testing"
)

func Test_application_SubmitConnection(t *testing.T) {
	type fields struct {
		config       ApplicationConfig
		routingLock  sync.RWMutex
		upstreamConn map[string]int
	}
	type args struct {
		client net.Conn
		rlm    RateLimitManager
	}

	RateLimitConfig := RateLimitManagerConfig{
		MaxOpenConnections:   5,
		MaxRateAmount:        5,
		MaxRatePeriodSeconds: 10,
	}

	upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalln("Could not open upstream server", err)
	}
	upstreamAddress := upstreamListener.Addr().String()
	log.Println("UT upstream running on", upstreamAddress)

	go func() {
		defer upstreamListener.Close()
		conn, err := upstreamListener.Accept()
		log.Println("Upstream accepted incoming")
		if err != nil {
			log.Fatalln("Could not accept incoming", err)
		}

		go func() {
			log.Println("Upstream disconnecting async")
			err = conn.Close()
			if err != nil {
				log.Fatalln("Could not close client conn", err)
			}
		}()
	}()

	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalln("Could not open proxy server", err)
	}

	log.Println("UT proxy running on", proxyListener.Addr().String())
	go func() {
		defer proxyListener.Close()
		_, acceptErr := proxyListener.Accept()
		log.Println("Proxy accepted incoming")
		if acceptErr != nil {
			log.Fatalln("Could not accept incoming", err)
		}
	}()

	clientConn, err := net.Dial("tcp", proxyListener.Addr().String())
	if err != nil {
		log.Fatalln("Could not connect to server", err)
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "simpleConnect",
			fields: fields{
				config: ApplicationConfig{
					Name: "ut",
					Upstreams: []UpstreamServer{
						{Address: upstreamAddress},
					},
				},
				routingLock:  sync.RWMutex{},
				upstreamConn: map[string]int{upstreamAddress: 0},
			},
			args: args{
				client: clientConn,
				rlm:    CreateRateLimitManager("ut", RateLimitConfig),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &application{
				config:       tt.fields.config,
				routingLock:  tt.fields.routingLock,
				upstreamConn: tt.fields.upstreamConn,
			}
			a.SubmitConnection(tt.args.client, tt.args.rlm)
		})
	}
}
