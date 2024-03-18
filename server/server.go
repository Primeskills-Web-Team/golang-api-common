package server

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/exception"
	"github.com/Primeskills-Web-Team/golang-server/primeskillsserver"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	eureka "github.com/xuanbo/eureka-client"
	"log"
	"os"
	"strconv"
)

func defaultConfig() (string, string) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	resource := os.Getenv("APP_NAME")
	return port, resource
}

func Run() {
	port, resource := defaultConfig()
	if port == "" {
		log.Fatalln("Port is undefined, please setup your port")
	}
	RunDefaultServer(port, resource, nil)
}

func InitServer(routes func(e *gin.Engine)) {
	port, resource := defaultConfig()
	RunDefaultServer(port, resource, routes)
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
