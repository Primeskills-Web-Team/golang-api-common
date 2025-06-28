package redis

import (
	"crypto/tls"
	"runtime"
	"testing"
)

func TestConfigDefault(t *testing.T) {
	tests := []struct {
		name     string
		config   []Config
		expected Config
	}{
		{
			name:     "No config provided",
			config:   []Config{},
			expected: ConfigDefault,
		},
		{
			name: "Empty config provided",
			config: []Config{
				{},
			},
			expected: Config{
				Host:             "127.0.0.1",
				Port:             6379,
				Username:         "",
				Password:         "",
				URL:              "",
				Database:         0,
				Reset:            false,
				TLSConfig:        nil,
				PoolSize:         10 * runtime.GOMAXPROCS(0),
				Addrs:            []string{},
				MasterName:       "",
				ClientName:       "",
				SentinelUsername: "",
				SentinelPassword: "",
			},
		},
		{
			name: "Partial config provided - only host",
			config: []Config{
				{
					Host: "localhost",
				},
			},
			expected: Config{
				Host:             "localhost",
				Port:             6379,
				Username:         "",
				Password:         "",
				URL:              "",
				Database:         0,
				Reset:            false,
				TLSConfig:        nil,
				PoolSize:         10 * runtime.GOMAXPROCS(0),
				Addrs:            []string{},
				MasterName:       "",
				ClientName:       "",
				SentinelUsername: "",
				SentinelPassword: "",
			},
		},
		{
			name: "Partial config provided - only port",
			config: []Config{
				{
					Port: 6380,
				},
			},
			expected: Config{
				Host:             "127.0.0.1",
				Port:             6380,
				Username:         "",
				Password:         "",
				URL:              "",
				Database:         0,
				Reset:            false,
				TLSConfig:        nil,
				PoolSize:         10 * runtime.GOMAXPROCS(0),
				Addrs:            []string{},
				MasterName:       "",
				ClientName:       "",
				SentinelUsername: "",
				SentinelPassword: "",
			},
		},
		{
			name: "Full config provided",
			config: []Config{
				{
					Host:             "redis.example.com",
					Port:             6380,
					Username:         "user",
					Password:         "pass",
					Database:         1,
					URL:              "redis://user:pass@redis.example.com:6380/1",
					Reset:            true,
					TLSConfig:        &tls.Config{},
					PoolSize:         20,
					Addrs:            []string{"redis1:6379", "redis2:6379"},
					MasterName:       "mymaster",
					ClientName:       "test-client",
					SentinelUsername: "sentinel-user",
					SentinelPassword: "sentinel-pass",
				},
			},
			expected: Config{
				Host:             "redis.example.com",
				Port:             6380,
				Username:         "user",
				Password:         "pass",
				Database:         1,
				URL:              "redis://user:pass@redis.example.com:6380/1",
				Reset:            true,
				TLSConfig:        &tls.Config{},
				PoolSize:         20,
				Addrs:            []string{"redis1:6379", "redis2:6379"},
				MasterName:       "mymaster",
				ClientName:       "test-client",
				SentinelUsername: "sentinel-user",
				SentinelPassword: "sentinel-pass",
			},
		},
		{
			name: "Zero port should use default",
			config: []Config{
				{
					Host: "localhost",
					Port: 0,
				},
			},
			expected: Config{
				Host:             "localhost",
				Port:             6379,
				Username:         "",
				Password:         "",
				URL:              "",
				Database:         0,
				Reset:            false,
				TLSConfig:        nil,
				PoolSize:         10 * runtime.GOMAXPROCS(0),
				Addrs:            []string{},
				MasterName:       "",
				ClientName:       "",
				SentinelUsername: "",
				SentinelPassword: "",
			},
		},
		{
			name: "Negative port should use default",
			config: []Config{
				{
					Host: "localhost",
					Port: -1,
				},
			},
			expected: Config{
				Host:             "localhost",
				Port:             6379,
				Username:         "",
				Password:         "",
				URL:              "",
				Database:         0,
				Reset:            false,
				TLSConfig:        nil,
				PoolSize:         10 * runtime.GOMAXPROCS(0),
				Addrs:            []string{},
				MasterName:       "",
				ClientName:       "",
				SentinelUsername: "",
				SentinelPassword: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configDefault(tt.config...)

			// Compare all fields
			if result.Host != tt.expected.Host {
				t.Errorf("Host = %v, want %v", result.Host, tt.expected.Host)
			}
			if result.Port != tt.expected.Port {
				t.Errorf("Port = %v, want %v", result.Port, tt.expected.Port)
			}
			if result.Username != tt.expected.Username {
				t.Errorf("Username = %v, want %v", result.Username, tt.expected.Username)
			}
			if result.Password != tt.expected.Password {
				t.Errorf("Password = %v, want %v", result.Password, tt.expected.Password)
			}
			if result.Database != tt.expected.Database {
				t.Errorf("Database = %v, want %v", result.Database, tt.expected.Database)
			}
			if result.URL != tt.expected.URL {
				t.Errorf("URL = %v, want %v", result.URL, tt.expected.URL)
			}
			if result.Reset != tt.expected.Reset {
				t.Errorf("Reset = %v, want %v", result.Reset, tt.expected.Reset)
			}
			if result.PoolSize != tt.expected.PoolSize {
				t.Errorf("PoolSize = %v, want %v", result.PoolSize, tt.expected.PoolSize)
			}
			if result.MasterName != tt.expected.MasterName {
				t.Errorf("MasterName = %v, want %v", result.MasterName, tt.expected.MasterName)
			}
			if result.ClientName != tt.expected.ClientName {
				t.Errorf("ClientName = %v, want %v", result.ClientName, tt.expected.ClientName)
			}
			if result.SentinelUsername != tt.expected.SentinelUsername {
				t.Errorf("SentinelUsername = %v, want %v", result.SentinelUsername, tt.expected.SentinelUsername)
			}
			if result.SentinelPassword != tt.expected.SentinelPassword {
				t.Errorf("SentinelPassword = %v, want %v", result.SentinelPassword, tt.expected.SentinelPassword)
			}

			// Check slice equality for Addrs
			if len(result.Addrs) != len(tt.expected.Addrs) {
				t.Errorf("Addrs length = %v, want %v", len(result.Addrs), len(tt.expected.Addrs))
			} else {
				for i, addr := range result.Addrs {
					if addr != tt.expected.Addrs[i] {
						t.Errorf("Addrs[%d] = %v, want %v", i, addr, tt.expected.Addrs[i])
					}
				}
			}

			// TLS Config comparison (both nil or both non-nil)
			if (result.TLSConfig == nil) != (tt.expected.TLSConfig == nil) {
				t.Errorf("TLSConfig nil status = %v, want %v", result.TLSConfig == nil, tt.expected.TLSConfig == nil)
			}
		})
	}
}

func TestConfigDefaultValues(t *testing.T) {
	// Test that ConfigDefault has expected values
	expected := Config{
		Host:             "127.0.0.1",
		Port:             6379,
		Username:         "",
		Password:         "",
		URL:              "",
		Database:         0,
		Reset:            false,
		TLSConfig:        nil,
		PoolSize:         10 * runtime.GOMAXPROCS(0),
		Addrs:            []string{},
		MasterName:       "",
		ClientName:       "",
		SentinelUsername: "",
		SentinelPassword: "",
	}

	if ConfigDefault.Host != expected.Host {
		t.Errorf("ConfigDefault.Host = %v, want %v", ConfigDefault.Host, expected.Host)
	}
	if ConfigDefault.Port != expected.Port {
		t.Errorf("ConfigDefault.Port = %v, want %v", ConfigDefault.Port, expected.Port)
	}
	if ConfigDefault.Username != expected.Username {
		t.Errorf("ConfigDefault.Username = %v, want %v", ConfigDefault.Username, expected.Username)
	}
	if ConfigDefault.Password != expected.Password {
		t.Errorf("ConfigDefault.Password = %v, want %v", ConfigDefault.Password, expected.Password)
	}
	if ConfigDefault.Database != expected.Database {
		t.Errorf("ConfigDefault.Database = %v, want %v", ConfigDefault.Database, expected.Database)
	}
	if ConfigDefault.URL != expected.URL {
		t.Errorf("ConfigDefault.URL = %v, want %v", ConfigDefault.URL, expected.URL)
	}
	if ConfigDefault.Reset != expected.Reset {
		t.Errorf("ConfigDefault.Reset = %v, want %v", ConfigDefault.Reset, expected.Reset)
	}
	if ConfigDefault.PoolSize != expected.PoolSize {
		t.Errorf("ConfigDefault.PoolSize = %v, want %v", ConfigDefault.PoolSize, expected.PoolSize)
	}
	if ConfigDefault.TLSConfig != expected.TLSConfig {
		t.Errorf("ConfigDefault.TLSConfig = %v, want %v", ConfigDefault.TLSConfig, expected.TLSConfig)
	}
}
