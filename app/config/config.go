// common/config/config.go
package config

type AppConfig struct {
	Port                string
	AppName             string
	ServiceDiscoveryURL string
	AlertEnabled        bool
	Environment         string
	Version             string
}
