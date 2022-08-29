package internal

import "github.com/danielepagano/teleport-int-load-balancer/lib/lbproxy"

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
