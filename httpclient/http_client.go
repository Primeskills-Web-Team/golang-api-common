package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// Config holds the configuration for the HTTP client
type Config struct {
	// Timeout specifies the maximum duration for an HTTP request.
	// If the request takes longer than this duration, it will be canceled.
	//
	// Default: 30 seconds
	Timeout time.Duration

	// MaxRetries specifies the maximum number of retries for a failed request.
	// If a request fails, it will be retried up to this number of times.
	//
	// Default: 3
	MaxRetries int

	// RetryWaitTime specifies the initial wait time before retrying a failed request.
	// The wait time will double with each retry, up to MaxRetryWaitTime.
	//
	// Default: 1 second
	RetryWaitTime time.Duration

	// MaxRetryWaitTime specifies the maximum wait time between retries.
	// If the wait time exceeds this duration, it will be capped at this value.
	//
	// Default: 10 seconds
	MaxRetryWaitTime time.Duration
}

var DefaultConfig = Config{
	Timeout:          30 * time.Second,
	MaxRetries:       3,
	RetryWaitTime:    1 * time.Second,
	MaxRetryWaitTime: 10 * time.Second,
}

// setConfig sets the HTTP client configuration.
func setConfig(config ...Config) Config {
	if len(config) == 0 {
		return DefaultConfig
	}

	// Override default config with provided configs
	cfg := config[0]

	// Set default values if not provided
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultConfig.Timeout
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = DefaultConfig.MaxRetries
	}
	if cfg.RetryWaitTime == 0 {
		cfg.RetryWaitTime = DefaultConfig.RetryWaitTime
	}
	if cfg.MaxRetryWaitTime == 0 {
		cfg.MaxRetryWaitTime = DefaultConfig.MaxRetryWaitTime
	}
	return cfg
}

type HttpClient struct {
	config Config
	client *http.Client
}

// New creates a new HttpClient with the provided configuration.
func New(config ...Config) HttpClient {
	cfg := setConfig(config...)
	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	return HttpClient{
		config: cfg,
		client: client,
	}
}

// Call executes an HTTP request with the specified method, URL, headers, and body.
func (h *HttpClient) Call(ctx context.Context, method string, url string, headers map[string]string, body interface{}, result interface{}) error {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		// supported method, no action needed
	default:
		return fmt.Errorf("unsupported HTTP method: %s", method)
	}

	var reqData []byte
	if body != nil {
		var err error
		reqData, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers if not provided
	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}

	for k, v := range headers {
		request.Header.Set(k, v)
	}

	// Log request
	log.Debug().
		Str("method", method).
		Str("url", url).
		Interface("headers", headers).
		Interface("body", body).
		Msg("Making HTTP request")

	var response *http.Response
	var retryCount int
	backoff := h.config.RetryWaitTime

	for retryCount <= h.config.MaxRetries {
		response, err = h.client.Do(request)
		if err == nil {
			break
		}

		retryCount++
		if retryCount > h.config.MaxRetries {
			return fmt.Errorf("failed after %d retries: %w", h.config.MaxRetries, err)
		}

		log.Warn().
			Err(err).
			Int("retry", retryCount).
			Dur("wait_time", backoff).
			Msg("Request failed, retrying...")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff = min(backoff*2, h.config.MaxRetryWaitTime)
		}
	}
	defer response.Body.Close()

	resBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response
	log.Debug().
		Int("status_code", response.StatusCode).
		Str("response", string(resBody)).
		Msg("Received HTTP response")

	// Check for error status codes
	if response.StatusCode >= 400 {
		return fmt.Errorf("request failed with status %d: %s", response.StatusCode, string(resBody))
	}

	if len(resBody) > 0 && result != nil {
		if err := json.Unmarshal(resBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
