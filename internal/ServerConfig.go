package internal

import "github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"

// GetStaticConfig is a placeholder source for configuration
// To test these apps easily (without security), open an echo server for each upstream, e.g. `ncat -l 1230 --keep-open --exec "/bin/cat"`
// you can then use nc to send data through proxy, e.g. `nc localhost 9001` would connect to app1, and you should see echos
// You can simulate EOF in both direction by using Ctrl-C on either nc (client) or ncat (server)
func GetStaticConfig() *ServerConfig {
	return &ServerConfig{
		Apps: []AppConfig{
			{
				AppId:     "app1",
				ProxyPort: "9001",
				Upstreams: []lbproxy.UpstreamServer{
					{Address: ":1230"},
					{Address: ":4560"},
				},
			},
			{
				AppId:     "app2",
				ProxyPort: "9002",
				Upstreams: []lbproxy.UpstreamServer{
					{Address: ":3210"},
					{Address: ":6540"},
				},
			},
		},
		Clients: []ClientConfig{
			{
				ClientId:      "one.com",
				AllowedAppIds: []string{"app1"},
			},
			{
				ClientId:      "two.com",
				AllowedAppIds: []string{"app2"},
			},
			{
				ClientId:      "all.com",
				AllowedAppIds: []string{"app1", "app2"},
			},
		},
		DefaultRateLimitConfig: lbproxy.RateLimitManagerConfig{
			MaxOpenConnections:   3,
			MaxRateAmount:        2,
			MaxRatePeriodSeconds: 5,
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
	Clients                []ClientConfig
	DefaultRateLimitConfig lbproxy.RateLimitManagerConfig
}

type AppConfig struct {
	AppId     string
	ProxyPort string
	Upstreams []lbproxy.UpstreamServer
}

type ClientConfig struct {
	ClientId      string
	AllowedAppIds []string
}
