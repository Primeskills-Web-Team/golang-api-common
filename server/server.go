package server

import (
	"fmt"
	"net/http"
	"time"

	"log"
	"os"
	"strconv"

	"github.com/Primeskills-Web-Team/golang-api-common/exception"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka/integration/config"
	"github.com/Primeskills-Web-Team/golang-server/primeskillsserver"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	eureka "github.com/xuanbo/eureka-client"
	// ✅ Add your imports
	// "github.com/Primeskills-Web-Team/golang-api-common/kafka/config"           // Adjust to your actual path
	// "github.com/Primeskills-Web-Team/golang-api-common/pkg/circuitbreaker"
)

// ✅ Global Kafka instance
var kafkaConfig *config.KafkaConfig

func defaultConfig() (string, string) {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    resource := os.Getenv("APP_NAME")
    return port, resource
}

// ✅ Initialize Kafka with DLQ
func initKafka() {
    logrus.Info("Initializing Kafka with DLQ...")
    
    // ✅ Load Kafka configuration from environment
    kafkaConfig = &config.KafkaConfig{
        Address:  []string{os.Getenv("KAFKA_BROKERS")}, // e.g., "localhost:9092,localhost:9093"
        Username: os.Getenv("KAFKA_USERNAME"),
        Password: os.Getenv("KAFKA_PASSWORD"),
        SlackWebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
    }
    
    // ✅ Set default if KAFKA_BROKERS not set
    if kafkaConfig.Address[0] == "" {
        kafkaConfig.Address = []string{"localhost:9092"}
        logrus.Warn("KAFKA_BROKERS not set, using default: localhost:9092")
    }
    
    // ✅ Initialize DLQ with custom configuration
    dlqConfig := config.DLQConfig{
        Enabled:              true,
        MaxRetries:           getEnvAsInt("DLQ_MAX_RETRIES", 3),
        RetryDelay:           getEnvAsDuration("DLQ_RETRY_DELAY", time.Second*2),
        MaxRetryDelay:        getEnvAsDuration("DLQ_MAX_RETRY_DELAY", time.Minute*5),
        DeadLetterSuffix:     getEnvAsString("DLQ_SUFFIX", "-dead-letter"),
        EnableCircuitBreaker: getEnvAsBool("DLQ_CIRCUIT_BREAKER_ENABLED", true),
        EnableDeduplication:  getEnvAsBool("DLQ_DEDUPLICATION_ENABLED", true),
        CircuitBreakerConfig: config.CircuitBreakerConfig{
            MaxFailures:  getEnvAsInt("CB_MAX_FAILURES", 5),
            ResetTimeout: getEnvAsDuration("CB_RESET_TIMEOUT", time.Minute*2),
        },
    }
    
    kafkaConfig.InitializeWithDLQ(dlqConfig)
    
    // ✅ Test DLQ functionality on startup
    if getEnvAsBool("DLQ_TEST_ON_STARTUP", false) {
        go func() {
            time.Sleep(time.Second * 5) // Wait for server to start
            if err := kafkaConfig.TestDLQ(); err != nil {
                logrus.WithError(err).Error("DLQ test failed on startup")
            }
        }()
    }
    
    logrus.WithFields(logrus.Fields{
        "kafka_brokers":     kafkaConfig.Address,
        "dlq_enabled":       dlqConfig.Enabled,
        "circuit_breaker":   dlqConfig.EnableCircuitBreaker,
        "max_retries":       dlqConfig.MaxRetries,
    }).Info("Kafka DLQ initialized successfully")
}

// ✅ Setup monitoring routes
func setupMonitoringRoutes(e *gin.Engine) {
    monitoring := e.Group("/monitoring")
    {
        // ✅ Health check endpoint
        monitoring.GET("/health", healthCheckHandler)
        
        // ✅ DLQ specific health
        monitoring.GET("/dlq/health", dlqHealthHandler)
        
        // ✅ DLQ metrics
        monitoring.GET("/dlq/metrics", dlqMetricsHandler)
        
        // ✅ Circuit breaker status
        monitoring.GET("/dlq/circuit-breaker", circuitBreakerHandler)
        
        // ✅ Test DLQ functionality
        monitoring.POST("/dlq/test", dlqTestHandler)
        
        // ✅ Reset circuit breaker
        monitoring.POST("/dlq/circuit-breaker/reset", resetCircuitBreakerHandler)
        
        // ✅ Send test Slack alert
        monitoring.POST("/dlq/test-alert", testSlackAlertHandler)
    }
}

// ✅ Health check handler
func healthCheckHandler(c *gin.Context) {
    health := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now(),
        "service":   os.Getenv("APP_NAME"),
        "version":   os.Getenv("VERSION"),
    }
    
    if kafkaConfig != nil {
        dlqHealth := kafkaConfig.GetDLQHealthStatus()
        health["dlq"] = dlqHealth
        
        // ✅ Determine overall health based on DLQ status
        if circuitBreakerData, ok := dlqHealth["circuit_breaker"].(map[string]interface{}); ok {
            if isHealthy, ok := circuitBreakerData["is_healthy"].(bool); ok && !isHealthy {
                health["status"] = "degraded"
            }
        }
    }
    
    c.JSON(http.StatusOK, health)
}

// ✅ DLQ health handler
func dlqHealthHandler(c *gin.Context) {
    if kafkaConfig == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "Kafka DLQ not initialized",
        })
        return
    }
    
    health := kafkaConfig.GetDLQHealthStatus()
    
    // ✅ Determine HTTP status based on health
    status := http.StatusOK
    if circuitBreakerData, ok := health["circuit_breaker"].(map[string]interface{}); ok {
        if state, ok := circuitBreakerData["state"].(string); ok && state == "OPEN" {
            status = http.StatusServiceUnavailable
        }
    }
    
    c.JSON(status, health)
}

// ✅ DLQ metrics handler
func dlqMetricsHandler(c *gin.Context) {
    if kafkaConfig == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "Kafka DLQ not initialized",
        })
        return
    }
    
    metrics := kafkaConfig.GetDLQMetrics()
    c.JSON(http.StatusOK, metrics)
}

// ✅ Circuit breaker status handler
func circuitBreakerHandler(c *gin.Context) {
    if kafkaConfig == nil || kafkaConfig.GetCircuitBreaker() == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "Circuit breaker not initialized",
        })
        return
    }
    
    cb := kafkaConfig.GetCircuitBreaker()
    metrics := cb.GetMetrics()
    
    response := map[string]interface{}{
        "circuit_breaker": metrics,
        "additional_info": map[string]interface{}{
            "failure_rate": cb.GetFailureRate(),
            "success_rate": cb.GetSuccessRate(),
            "is_healthy":   cb.IsHealthy(),
            "uptime":       cb.GetUptime().String(),
        },
    }
    
    c.JSON(http.StatusOK, response)
}

// ✅ Test DLQ handler
func dlqTestHandler(c *gin.Context) {
    if kafkaConfig == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "Kafka DLQ not initialized",
        })
        return
    }
    
    logrus.Info("Manual DLQ test triggered via API")
    
    if err := kafkaConfig.TestDLQ(); err != nil {
        logrus.WithError(err).Error("DLQ test failed")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "DLQ test failed",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message":   "DLQ test initiated successfully",
        "timestamp": time.Now(),
        "note":      "Check logs and monitoring for test results",
    })
}

// ✅ Reset circuit breaker handler
func resetCircuitBreakerHandler(c *gin.Context) {
    if kafkaConfig == nil || kafkaConfig.GetCircuitBreaker() == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "Circuit breaker not initialized",
        })
        return
    }
    
    cb := kafkaConfig.GetCircuitBreaker()
    oldState := cb.GetStateName()
    cb.Reset()
    newState := cb.GetStateName()
    
    logrus.WithFields(logrus.Fields{
        "old_state": oldState,
        "new_state": newState,
    }).Info("Circuit breaker manually reset via API")
    
    c.JSON(http.StatusOK, gin.H{
        "message":   "Circuit breaker reset successfully",
        "old_state": oldState,
        "new_state": newState,
        "timestamp": time.Now(),
    })
}

// ✅ Test Slack alert handler
func testSlackAlertHandler(c *gin.Context) {
    if kafkaConfig == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "Kafka DLQ not initialized",
        })
        return
    }
    
    logrus.Info("Manual Slack alert test triggered via API")
    
    if err := kafkaConfig.SendTestSlackAlert(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to send test Slack alert",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message":   "Test Slack alert sent successfully",
        "timestamp": time.Now(),
    })
}

func Run() {
    port, resource := defaultConfig()
    if port == "" {
        log.Fatalln("Port is undefined, please setup your port")
    }
    
    // ✅ Initialize Kafka DLQ
    initKafka()
    
    // ✅ Setup routes with monitoring
    routes := func(e *gin.Engine) {
        setupMonitoringRoutes(e)
        // Add your other routes here
    }
    
    RunDefaultServer(port, resource, routes)
}

func InitServer(routes func(e *gin.Engine)) {
    port, resource := defaultConfig()
    
    // ✅ Initialize Kafka DLQ
    initKafka()
    
    // ✅ Combine monitoring routes with user routes
    combinedRoutes := func(e *gin.Engine) {
        setupMonitoringRoutes(e)
        if routes != nil {
            routes(e)
        }
    }
    
    RunDefaultServer(port, resource, combinedRoutes)
}

func RunDefaultServer(port string, resource string, routes func(e *gin.Engine)) {
    srv := primeskillsserver.NewPrimeskillsServer()
    srv.SetException(exception.ErrorHandler)
    srv.SetStatusMethodNotAllowed(func(c *gin.Context) {
        panic(exception.NewMethodNotAllowedError(fmt.Sprintf("Path %s with methode is not allowed", c.Request.RequestURI)))
    })
    srv.SetStatusNotFound(func(c *gin.Context) {
        panic(exception.NewNotFoundErrorError(fmt.Sprintf("Path %s not found", c.Request.RequestURI)))
    })

    if routes != nil {
        srv.SetRouters(routes)
    }
    
    heartbeat(port)
    
    // ✅ Log monitoring endpoints
    logrus.WithFields(logrus.Fields{
        "port": port,
        "monitoring_endpoints": []string{
            "/monitoring/health",
            "/monitoring/dlq/health",
            "/monitoring/dlq/metrics",
            "/monitoring/dlq/circuit-breaker",
            "/monitoring/dlq/test",
            "/monitoring/dlq/circuit-breaker/reset",
            "/monitoring/dlq/test-alert",
        },
    }).Info("Server starting with DLQ monitoring endpoints")
    
    srv.RunServer(port, resource)
}

func heartbeat(port string) {
    logrus.Infoln("Register to eureka")
    portInt, _ := strconv.Atoi(port)
    eurekaConfig := eureka.Config{
        DefaultZone:           os.Getenv("SERVICE_DISCOVERY_URL"),
        App:                   os.Getenv("APP_NAME"),
        Port:                  portInt,
        RenewalIntervalInSecs: 10,
        DurationInSecs:        30,
        Metadata: map[string]interface{}{
            "VERSION":              os.Getenv("VERSION"),
            "NODE_GROUP_ID":        0,
            "PRODUCT_CODE":         "DEFAULT",
            "PRODUCT_VERSION_CODE": "DEFAULT",
            "PRODUCT_ENV_CODE":     "DEFAULT",
            "SERVICE_VERSION_CODE": "DEFAULT",
            // ✅ Add DLQ metadata
            "DLQ_ENABLED":          "true",
            "CIRCUIT_BREAKER":      "true",
        },
    }

    if os.Getenv("HOST_NAME_APP") != "" {
        eurekaConfig.HostName = os.Getenv("HOST_NAME_APP")
    }

    if os.Getenv("IP_APP") != "" {
        eurekaConfig.IP = os.Getenv("IP_APP")
    }

    client := eureka.NewClient(&eurekaConfig)
    client.Start()
}

// ✅ Helper functions for environment variables
func getEnvAsString(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if boolValue, err := strconv.ParseBool(value); err == nil {
            return boolValue
        }
    }
    return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}

// ✅ Getter for Kafka config (for use in other parts of your app)
func GetKafkaConfig() *config.KafkaConfig {
    return kafkaConfig
}