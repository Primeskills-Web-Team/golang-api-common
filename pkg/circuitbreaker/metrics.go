package circuitbreaker

import (
    "sync/atomic"
    "time"
)

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() Metrics {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    
    metrics := Metrics{
        Name:           cb.name,
        State:          cb.getStateName(),
        FailureCount:   cb.failureCount,
        SuccessCount:   cb.successCount,
        TotalRequests:  atomic.LoadInt64(&cb.totalRequests),
        TotalFailures:  atomic.LoadInt64(&cb.totalFailures),
        TotalSuccesses: atomic.LoadInt64(&cb.totalSuccesses),
    }
    
    // ✅ Add timestamps if available
    if !cb.lastFailureTime.IsZero() {
        metrics.LastFailureTime = &cb.lastFailureTime
    }
    
    if !cb.lastSuccessTime.IsZero() {
        metrics.LastSuccessTime = &cb.lastSuccessTime
    }
    
    return metrics
}

// GetFailureRate returns current failure rate (0.0 to 1.0)
func (cb *CircuitBreaker) GetFailureRate() float64 {
    totalReq := atomic.LoadInt64(&cb.totalRequests)
    if totalReq == 0 {
        return 0.0
    }
    
    totalFail := atomic.LoadInt64(&cb.totalFailures)
    return float64(totalFail) / float64(totalReq)
}

// GetSuccessRate returns current success rate (0.0 to 1.0)
func (cb *CircuitBreaker) GetSuccessRate() float64 {
    return 1.0 - cb.GetFailureRate()
}

// IsHealthy returns true if circuit breaker is healthy
func (cb *CircuitBreaker) IsHealthy() bool {
    return cb.IsClosed() && cb.GetFailureRate() < 0.5 // Less than 50% failure rate
}

// GetUptime returns how long the circuit breaker has been in current state
func (cb *CircuitBreaker) GetUptime() time.Duration {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    
    switch cb.state {
    case StateOpen:
        if !cb.lastFailureTime.IsZero() {
            return time.Since(cb.lastFailureTime)
        }
    case StateClosed:
        if !cb.lastSuccessTime.IsZero() {
            return time.Since(cb.lastSuccessTime)
        }
    }
    
    return 0
}