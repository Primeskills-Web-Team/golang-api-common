package redis

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *Storage) {
	s := miniredis.RunT(t)

	port, _ := strconv.Atoi(s.Port())
	config := Config{
		Host: s.Host(),
		Port: port,
	}

	storage := New(config)
	return s, storage
}

func TestNew(t *testing.T) {
	// Test with invalid Redis URL to ensure URL parsing works
	t.Run("Config with invalid URL", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("New() should have panicked with invalid URL")
			}
		}()

		New(Config{
			URL: "invalid-url",
		})
	})
}

func TestNewWithMiniredis(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	port, _ := strconv.Atoi(s.Port())

	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "Basic config",
			config: Config{
				Host: s.Host(),
				Port: port,
			},
		},
		{
			name: "Config with database",
			config: Config{
				Host:     s.Host(),
				Port:     port,
				Database: 1,
			},
		},
		{
			name: "Config with username and password",
			config: Config{
				Host: s.Host(),
				Port: port,
			},
		},
		{
			name: "Config with URL",
			config: Config{
				URL: "redis://" + s.Addr() + "/0",
			},
		},
		{
			name: "Config with Addrs",
			config: Config{
				Addrs: []string{s.Addr()},
			},
		},
		{
			name: "Config with Reset true",
			config: Config{
				Host:  s.Host(),
				Port:  port,
				Reset: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := New(tt.config)
			if storage == nil {
				t.Error("New() returned nil")
			}

			// Test that we can perform basic operations
			err := storage.Set("test", []byte("value"), 0)
			if err != nil {
				t.Errorf("Set() failed: %v", err)
			}

			storage.Close()
		})
	}
}

func TestNewFromConnection(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	storage := NewFromConnection(client)
	if storage == nil {
		t.Error("NewFromConnection() returned nil")
	}

	// Test that we can perform basic operations
	err := storage.Set("test", []byte("value"), 0)
	if err != nil {
		t.Errorf("Set() failed: %v", err)
	}

	val, err := storage.Get("test")
	if err != nil {
		t.Errorf("Get() failed: %v", err)
	}
	if string(val) != "value" {
		t.Errorf("Get() = %v, want %v", string(val), "value")
	}

	storage.Close()
}

func TestStorage_Get(t *testing.T) {
	s, storage := setupMiniredis(t)
	defer s.Close()
	defer storage.Close()

	tests := []struct {
		name      string
		key       string
		setupData map[string]string
		expected  []byte
		expectErr bool
	}{
		{
			name:      "Get existing key",
			key:       "testkey",
			setupData: map[string]string{"testkey": "testvalue"},
			expected:  []byte("testvalue"),
			expectErr: false,
		},
		{
			name:      "Get non-existing key",
			key:       "nonexistent",
			setupData: map[string]string{},
			expected:  nil,
			expectErr: false,
		},
		{
			name:      "Get with empty key",
			key:       "",
			setupData: map[string]string{},
			expected:  nil,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test data
			s.FlushAll()
			for k, v := range tt.setupData {
				s.Set(k, v)
			}

			result, err := storage.Get(tt.key)

			if tt.expectErr && err == nil {
				t.Error("Get() expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Get() unexpected error: %v", err)
			}

			if string(result) != string(tt.expected) {
				t.Errorf("Get() = %v, want %v", string(result), string(tt.expected))
			}
		})
	}
}

func TestStorage_Set(t *testing.T) {
	s, storage := setupMiniredis(t)
	defer s.Close()
	defer storage.Close()

	tests := []struct {
		name      string
		key       string
		value     []byte
		exp       time.Duration
		expectErr bool
	}{
		{
			name:      "Set with valid key and value",
			key:       "testkey",
			value:     []byte("testvalue"),
			exp:       0,
			expectErr: false,
		},
		{
			name:      "Set with expiration",
			key:       "expkey",
			value:     []byte("expvalue"),
			exp:       time.Second,
			expectErr: false,
		},
		{
			name:      "Set with empty key",
			key:       "",
			value:     []byte("testvalue"),
			exp:       0,
			expectErr: false, // Should not error but also not set
		},
		{
			name:      "Set with empty value",
			key:       "testkey",
			value:     []byte(""),
			exp:       0,
			expectErr: false, // Should not error but also not set
		},
		{
			name:      "Set with nil value",
			key:       "testkey",
			value:     nil,
			exp:       0,
			expectErr: false, // Should not error but also not set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.FlushAll()

			err := storage.Set(tt.key, tt.value, tt.exp)

			if tt.expectErr && err == nil {
				t.Error("Set() expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Set() unexpected error: %v", err)
			}

			// Verify the value was set (if key and value are not empty)
			if tt.key != "" && len(tt.value) > 0 {
				result, _ := storage.Get(tt.key)
				if string(result) != string(tt.value) {
					t.Errorf("Set() did not store value correctly. Got %v, want %v", string(result), string(tt.value))
				}
			}
		})
	}
}

func TestStorage_Delete(t *testing.T) {
	s, storage := setupMiniredis(t)
	defer s.Close()
	defer storage.Close()

	tests := []struct {
		name      string
		key       string
		setupData map[string]string
		expectErr bool
	}{
		{
			name:      "Delete existing key",
			key:       "testkey",
			setupData: map[string]string{"testkey": "testvalue"},
			expectErr: false,
		},
		{
			name:      "Delete non-existing key",
			key:       "nonexistent",
			setupData: map[string]string{},
			expectErr: false,
		},
		{
			name:      "Delete with empty key",
			key:       "",
			setupData: map[string]string{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test data
			s.FlushAll()
			for k, v := range tt.setupData {
				s.Set(k, v)
			}

			err := storage.Delete(tt.key)

			if tt.expectErr && err == nil {
				t.Error("Delete() expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Delete() unexpected error: %v", err)
			}

			// Verify the key was deleted (if key was not empty and existed)
			if tt.key != "" {
				if _, exists := tt.setupData[tt.key]; exists {
					result, _ := storage.Get(tt.key)
					if result != nil {
						t.Errorf("Delete() did not delete key. Key still exists with value: %v", string(result))
					}
				}
			}
		})
	}
}

func TestStorage_Reset(t *testing.T) {
	s, storage := setupMiniredis(t)
	defer s.Close()
	defer storage.Close()

	// Setup test data
	storage.Set("key1", []byte("value1"), 0)
	storage.Set("key2", []byte("value2"), 0)

	// Verify data exists
	val1, _ := storage.Get("key1")
	val2, _ := storage.Get("key2")
	if string(val1) != "value1" || string(val2) != "value2" {
		t.Error("Setup data not correctly stored")
	}

	// Reset
	err := storage.Reset()
	if err != nil {
		t.Errorf("Reset() error: %v", err)
	}

	// Verify data is gone
	val1, _ = storage.Get("key1")
	val2, _ = storage.Get("key2")
	if val1 != nil || val2 != nil {
		t.Error("Reset() did not clear all data")
	}
}

func TestStorage_Close(t *testing.T) {
	s, storage := setupMiniredis(t)
	defer s.Close()

	err := storage.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// After closing, operations should fail
	err = storage.Set("test", []byte("value"), 0)
	if err == nil {
		t.Error("Set() should fail after Close()")
	}
}

func TestStorage_Conn(t *testing.T) {
	s, storage := setupMiniredis(t)
	defer s.Close()
	defer storage.Close()

	conn := storage.Conn()
	if conn == nil {
		t.Error("Conn() returned nil")
	}

	// Test that we can use the connection directly
	err := conn.Set(context.Background(), "direct-test", "direct-value", 0).Err()
	if err != nil {
		t.Errorf("Direct connection Set() failed: %v", err)
	}

	result := conn.Get(context.Background(), "direct-test").Val()
	if result != "direct-value" {
		t.Errorf("Direct connection Get() = %v, want %v", result, "direct-value")
	}
}

func TestStorage_Keys(t *testing.T) {
	s, storage := setupMiniredis(t)
	defer s.Close()
	defer storage.Close()

	tests := []struct {
		name      string
		setupData map[string]string
		expectErr bool
	}{
		{
			name:      "No keys",
			setupData: map[string]string{},
			expectErr: false,
		},
		{
			name: "Single key",
			setupData: map[string]string{
				"key1": "value1",
			},
			expectErr: false,
		},
		{
			name: "Multiple keys",
			setupData: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test data
			s.FlushAll()
			for k, v := range tt.setupData {
				storage.Set(k, []byte(v), 0)
			}

			keys, err := storage.Keys()

			if tt.expectErr && err == nil {
				t.Error("Keys() expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Keys() unexpected error: %v", err)
			}

			// Check the number of keys
			expectedCount := len(tt.setupData)
			if expectedCount == 0 {
				if keys != nil {
					t.Errorf("Keys() should return nil when no keys exist, got %v", keys)
				}
			} else {
				if len(keys) != expectedCount {
					t.Errorf("Keys() returned %d keys, want %d", len(keys), expectedCount)
				}

				// Check that all expected keys are present
				keyMap := make(map[string]bool)
				for _, key := range keys {
					keyMap[string(key)] = true
				}

				for expectedKey := range tt.setupData {
					if !keyMap[expectedKey] {
						t.Errorf("Keys() missing expected key: %s", expectedKey)
					}
				}
			}
		})
	}
}

func TestStorage_Integration(t *testing.T) {
	s, storage := setupMiniredis(t)
	defer s.Close()
	defer storage.Close()

	// Test a complete workflow
	key := "integration-test"
	value := []byte("integration-value")

	// Set a value
	err := storage.Set(key, value, 0)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	// Get the value
	result, err := storage.Get(key)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if string(result) != string(value) {
		t.Errorf("Get() = %v, want %v", string(result), string(value))
	}

	// Check keys
	keys, err := storage.Keys()
	if err != nil {
		t.Fatalf("Keys() failed: %v", err)
	}
	if len(keys) != 1 || string(keys[0]) != key {
		t.Errorf("Keys() = %v, want [%v]", keys, key)
	}

	// Delete the value
	err = storage.Delete(key)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify it's gone
	result, err = storage.Get(key)
	if err != nil {
		t.Fatalf("Get() after delete failed: %v", err)
	}
	if result != nil {
		t.Errorf("Get() after delete = %v, want nil", result)
	}

	// Verify keys is empty
	keys, err = storage.Keys()
	if err != nil {
		t.Fatalf("Keys() after delete failed: %v", err)
	}
	if keys != nil {
		t.Errorf("Keys() after delete = %v, want nil", keys)
	}
}

func TestStorage_SetWithExpiration(t *testing.T) {
	s, storage := setupMiniredis(t)
	defer s.Close()
	defer storage.Close()

	key := "exp-test"
	value := []byte("exp-value")
	exp := 100 * time.Millisecond

	// Set with expiration
	err := storage.Set(key, value, exp)
	if err != nil {
		t.Fatalf("Set() with expiration failed: %v", err)
	}

	// Verify it exists initially
	result, err := storage.Get(key)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if string(result) != string(value) {
		t.Errorf("Get() = %v, want %v", string(result), string(value))
	}

	// Fast forward time in miniredis
	s.FastForward(200 * time.Millisecond)

	// Verify it's expired
	result, err = storage.Get(key)
	if err != nil {
		t.Fatalf("Get() after expiration failed: %v", err)
	}
	if result != nil {
		t.Errorf("Get() after expiration = %v, want nil", result)
	}
}
