package server

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Primeskills-Web-Team/golang-api-common/exception"
	"github.com/Primeskills-Web-Team/golang-api-common/kafka/integration/config"
	commonConfig "github.com/Primeskills-Web-Team/golang-api-common/app/config"
	"github.com/Primeskills-Web-Team/golang-server/primeskillsserver"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	eureka "github.com/xuanbo/eureka-client"
)

var kafkaConfig *config.KafkaConfig
var appConfig *commonConfig.AppConfig

// loadEnvConfig initializes basic config from environment variables
func loadEnvConfig() (port, appName string) {
	port = os.Getenv("PORT")
	if port == "" {
		log.Fatalln("PORT is undefined. Please set it in the environment.")
	}
	appName = os.Getenv("APP_NAME")
	return
}

// initKafka initializes Kafka configuration
func initKafka() {
	logrus.Info("Initializing Kafka with DLQ...")

	kafkaConfig = &config.KafkaConfig{
		Address:         []string{os.Getenv("KAFKA_BROKERS")},
		Username:        os.Getenv("KAFKA_USERNAME"),
		Password:        os.Getenv("KAFKA_PASSWORD"),
		SlackWebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		AlertEnabled:    os.Getenv("ALERT_ENABLED") == "true",
	}

	if kafkaConfig.Address[0] == "" {
		logrus.Warn("KAFKA_BROKERS not set")
	}
}

// initAppConfig initializes application configuration
func initAppConfig() {
    logrus.Info("Initializing application configuration...")

    appConfig = &commonConfig.AppConfig{
        Port:                os.Getenv("PORT"),
        AppName:             os.Getenv("APP_NAME"),
        ServiceDiscoveryURL: os.Getenv("SERVICE_DISCOVERY_URL"),
        AlertEnabled:        os.Getenv("ALERT_ENABLED") == "true",
        Environment:         os.Getenv("ENVIRONMENT"),
    }

    if appConfig.Port == "" {
        logrus.Fatal("PORT is not set in environment variables")
    }
    if appConfig.AppName == "" {
        logrus.Fatal("APP_NAME is not set in environment variables")
    }
}

// registerToEureka handles Eureka service registration
func registerToEureka(port string) {
	logrus.Info("Registering to Eureka...")

	portInt, err := strconv.Atoi(port)
	if err != nil {
		logrus.WithError(err).Fatal("Invalid PORT value, must be numeric")
	}

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
			"DLQ_ENABLED":          "true",
		},
	}

	if host := os.Getenv("HOST_NAME_APP"); host != "" {
		eurekaConfig.HostName = host
	}
	if ip := os.Getenv("IP_APP"); ip != "" {
		eurekaConfig.IP = ip
	}

	client := eureka.NewClient(&eurekaConfig)
	client.Start()
}

// Run is the default entry point for the server with internal routes only
func Run() {
	RunWithRoutes(nil)
}

// InitServer allows running with custom routes
func InitServer(customRoutes func(e *gin.Engine)) {
	RunWithRoutes(customRoutes)
}

// RunWithRoutes sets up the server and runs it
func RunWithRoutes(customRoutes func(e *gin.Engine)) {
	port, appName := loadEnvConfig()
	initKafka()
    initAppConfig()

	server := primeskillsserver.NewPrimeskillsServer()
	server.SetException(exception.ErrorHandler)

	server.SetStatusMethodNotAllowed(func(c *gin.Context) {
		panic(exception.NewMethodNotAllowedError(fmt.Sprintf("Path %s with method is not allowed", c.Request.RequestURI)))
	})

	server.SetStatusNotFound(func(c *gin.Context) {
		panic(exception.NewNotFoundErrorError(fmt.Sprintf("Path %s not found", c.Request.RequestURI)))
	})

	if customRoutes != nil {
		server.SetRouters(customRoutes)
	}

	registerToEureka(port)

	server.RunServer(port, appName)
}
