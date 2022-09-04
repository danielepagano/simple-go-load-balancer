package internal

import "github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"

// GetStaticConfig is a placeholder source for configuration
func GetStaticConfig() *ServerConfig {
	return &ServerConfig{
		Apps: []AppConfig{
			// Tests proxying to a remote http server (best with appropriately high rate limits)
			{
				AppId:     "httpbin",
				ProxyPort: "9001",
				Upstreams: []lbproxy.UpstreamServer{
					{Address: "eu.httpbin.org:80"},
					{Address: "httpbin.org:80"},
				},
			},
			// Open an echo server for each upstream, e.g. `ncat -l 9098 --keep-open --exec "/bin/cat"`;
			// you can then use `nc localhost 9002` to send data through proxy, and you should see echos
			// You can simulate EOF in both direction by using Ctrl-C on either nc (client) or ncat (server)
			{
				AppId:     "echo",
				ProxyPort: "9002",
				Upstreams: []lbproxy.UpstreamServer{
					{Address: ":9098"},
					{Address: ":9099"},
				},
			},
		},
		Clients: map[string][]string{
			"one.com":   {"httpbin"},
			"two.com":   {"echo"},
			"localhost": {"httpbin", "echo"},
		},
		DefaultRateLimitConfig: lbproxy.RateLimitManagerConfig{
			MaxOpenConnections:   5,
			MaxRateAmount:        5,
			MaxRatePeriodSeconds: 10,
		},
		SecurityConfig: ServerSecurityConfig{
			EnableMutualTLS: false,
			CACommonName:    "localhost",
			CertFilePath:    "",
			KeyFilePath:     "",
		},
	}
}

func (c *AppConfig) ToApplicationConfig() lbproxy.ApplicationConfig {
	return lbproxy.ApplicationConfig{
		Name:      c.AppId,
		Upstreams: c.Upstreams,
	}
}

type ServerConfig struct {
	Apps                   []AppConfig
	Clients                map[string][]string
	DefaultRateLimitConfig lbproxy.RateLimitManagerConfig
	SecurityConfig         ServerSecurityConfig
}

type ServerSecurityConfig struct {
	EnableMutualTLS bool // Master switch that turns off security to simplify testing in this sample project
	CACommonName    string
	CertFilePath    string
	KeyFilePath     string
}

type AppConfig struct {
	AppId     string
	ProxyPort string
	Upstreams []lbproxy.UpstreamServer
}
