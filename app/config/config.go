// common/config/config.go
package config

type AppConfig struct {
	Port                string
	AppName             string
	ServiceDiscoveryURL string
	Environment         string
	Version             string
}
