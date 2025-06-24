package circuitbreaker

import (
    "sync"
    "time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
    StateClosed CircuitState = iota
    StateOpen
    StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
    name            string
    maxFailures     int
    resetTimeout    time.Duration
    failureCount    int
    successCount    int           // ✅ Track success in half-open
    lastFailureTime time.Time
    lastSuccessTime time.Time
    state          CircuitState
    mutex          sync.RWMutex
    
    // ✅ ADD: Atomic counters for metrics
    totalRequests  int64         // ✅ Total requests processed
    totalFailures  int64         // ✅ Total failures
    totalSuccesses int64         // ✅ Total successes
    
    // ✅ Callbacks for monitoring
    onStateChange func(from, to CircuitState)
    onFailure     func(err error)
    onSuccess     func()
}

// Config for creating circuit breaker
type Config struct {
    Name                string
    MaxFailures         int
    ResetTimeout        time.Duration
    HalfOpenMaxRequests int           // ✅ Max requests in half-open state
    OnStateChange       func(from, to CircuitState)
    OnFailure          func(err error)
    OnSuccess          func()
}

// Metrics represents circuit breaker metrics
type Metrics struct {
    Name            string        `json:"name"`
    State           string        `json:"state"`
    FailureCount    int          `json:"failure_count"`
    SuccessCount    int          `json:"success_count"`
    LastFailureTime *time.Time   `json:"last_failure_time,omitempty"`
    LastSuccessTime *time.Time   `json:"last_success_time,omitempty"`
    TotalRequests   int64        `json:"total_requests"`
    TotalFailures   int64        `json:"total_failures"`
    TotalSuccesses  int64        `json:"total_successes"`
}