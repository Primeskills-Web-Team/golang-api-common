package circuitbreaker

import (
    "fmt"
    "sync/atomic"
    "time"
    
    "github.com/sirupsen/logrus"
)

// NewCircuitBreaker creates a new circuit breaker with default config
func NewCircuitBreaker(name string, maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
    return NewCircuitBreakerWithConfig(Config{
        Name:                name,
        MaxFailures:         maxFailures,
        ResetTimeout:        resetTimeout,
        HalfOpenMaxRequests: 3, // Default: allow 3 requests in half-open
    })
}

// NewCircuitBreakerWithConfig creates a new circuit breaker with custom config
func NewCircuitBreakerWithConfig(config Config) *CircuitBreaker {
    cb := &CircuitBreaker{
        name:            config.Name,
        maxFailures:     config.MaxFailures,
        resetTimeout:    config.ResetTimeout,
        state:          StateClosed,
        onStateChange:  config.OnStateChange,
        onFailure:      config.OnFailure,
        onSuccess:      config.OnSuccess,
    }
    
    logrus.WithFields(logrus.Fields{
        "name":         config.Name,
        "max_failures": config.MaxFailures,
        "reset_timeout": config.ResetTimeout,
    }).Info("Circuit breaker initialized")
    
    return cb
}

// Call executes the given operation with circuit breaker protection
func (cb *CircuitBreaker) Call(operation func() error) error {
    // ✅ Increment total requests
    atomic.AddInt64(&cb.totalRequests, 1)
    
    cb.mutex.Lock()
    defer cb.mutex.Unlock()

    // ✅ Check current state
    if cb.state == StateOpen {
        if time.Since(cb.lastFailureTime) > cb.resetTimeout {
            cb.setState(StateHalfOpen)
            cb.successCount = 0 // Reset success counter for half-open
        } else {
            return fmt.Errorf("circuit breaker '%s' is OPEN", cb.name)
        }
    }

    // ✅ Execute operation
    err := operation()

    if err != nil {
        cb.recordFailure(err)
        return err
    }

    cb.recordSuccess()
    return nil
}

// recordFailure handles failure scenarios
func (cb *CircuitBreaker) recordFailure(err error) {
    cb.failureCount++
    cb.lastFailureTime = time.Now()
    atomic.AddInt64(&cb.totalFailures, 1)
    
    // ✅ Call failure callback
    if cb.onFailure != nil {
        go cb.onFailure(err)
    }

    switch cb.state {
    case StateHalfOpen:
        // ✅ Half-open failed → back to open
        cb.setState(StateOpen)
        logrus.WithFields(logrus.Fields{
            "circuit_breaker": cb.name,
            "error": err.Error(),
        }).Warn("Circuit breaker: HALF-OPEN → OPEN (test failed)")
        
    case StateClosed:
        if cb.failureCount >= cb.maxFailures {
            // ✅ Too many failures → open
            cb.setState(StateOpen)
            logrus.WithFields(logrus.Fields{
                "circuit_breaker": cb.name,
                "failure_count": cb.failureCount,
                "max_failures": cb.maxFailures,
            }).Warn("Circuit breaker: CLOSED → OPEN (max failures reached)")
        }
    }
}

// recordSuccess handles success scenarios
func (cb *CircuitBreaker) recordSuccess() {
    cb.lastSuccessTime = time.Now()
    atomic.AddInt64(&cb.totalSuccesses, 1)
    
    // ✅ Call success callback
    if cb.onSuccess != nil {
        go cb.onSuccess()
    }

    switch cb.state {
    case StateHalfOpen:
        cb.successCount++
        // ✅ Enough successes in half-open → close
        if cb.successCount >= 3 { // Configurable
            cb.failureCount = 0
            cb.setState(StateClosed)
            logrus.WithField("circuit_breaker", cb.name).Info("Circuit breaker: HALF-OPEN → CLOSED (recovered)")
        }
        
    case StateClosed:
        // ✅ Reset failure count on success
        if cb.failureCount > 0 {
            cb.failureCount = 0
            logrus.WithField("circuit_breaker", cb.name).Debug("Circuit breaker: failure count reset")
        }
    }
}

// setState changes the circuit breaker state and triggers callback
func (cb *CircuitBreaker) setState(newState CircuitState) {
    oldState := cb.state
    cb.state = newState
    
    // ✅ Trigger state change callback
    if cb.onStateChange != nil && oldState != newState {
        go cb.onStateChange(oldState, newState)
    }
}

// IsOpen returns true if circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    return cb.state == StateOpen
}

// IsClosed returns true if circuit breaker is closed
func (cb *CircuitBreaker) IsClosed() bool {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    return cb.state == StateClosed
}

// IsHalfOpen returns true if circuit breaker is half-open
func (cb *CircuitBreaker) IsHalfOpen() bool {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    return cb.state == StateHalfOpen
}

// GetState returns current state
func (cb *CircuitBreaker) GetState() CircuitState {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    return cb.state
}

// GetStateName returns current state as string
func (cb *CircuitBreaker) GetStateName() string {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    return cb.getStateName()
}

func (cb *CircuitBreaker) getStateName() string {
    switch cb.state {
    case StateClosed:
        return "CLOSED"
    case StateOpen:
        return "OPEN"
    case StateHalfOpen:
        return "HALF-OPEN"
    default:
        return "UNKNOWN"
    }
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    oldState := cb.state
    cb.failureCount = 0
    cb.successCount = 0
    cb.setState(StateClosed)
    
    logrus.WithFields(logrus.Fields{
        "circuit_breaker": cb.name,
        "old_state": cb.getStateNameForState(oldState),
    }).Info("Circuit breaker manually reset")
}

func (cb *CircuitBreaker) getStateNameForState(state CircuitState) string {
    switch state {
    case StateClosed:
        return "CLOSED"
    case StateOpen:
        return "OPEN"
    case StateHalfOpen:
        return "HALF-OPEN"
    default:
        return "UNKNOWN"
    }
}

// Add total counters
var (
    totalRequests int64
    totalFailures int64
    totalSuccesses int64
)