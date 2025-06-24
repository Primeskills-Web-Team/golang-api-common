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
	Timeout          time.Duration
	MaxRetries       int
	RetryWaitTime    time.Duration
	MaxRetryWaitTime time.Duration
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:          30 * time.Second,
		MaxRetries:       3,
		RetryWaitTime:    1 * time.Second,
		MaxRetryWaitTime: 10 * time.Second,
	}
}

type HttpClient interface {
	Call(ctx context.Context, method string, url string, headers map[string]string, body interface{}, result interface{}) error
}

type httpClient struct {
	client *http.Client
	config *Config
}

// Call executes an HTTP request with the specified method, URL, headers, and body.
func (h *httpClient) Call(ctx context.Context, method string, url string, headers map[string]string, body interface{}, result interface{}) error {
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

// NewHttpClient creates a new instance of HttpClient with the provided configuration.
func NewHttpClient(config *Config) HttpClient {
	if config == nil {
		config = DefaultConfig()
	}

	return &httpClient{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}
}
