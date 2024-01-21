package server

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/exception"
	"github.com/Primeskills-Web-Team/golang-server/primeskillsserver"
	"github.com/gin-gonic/gin"
	"log"
)

func Run(port string, resource string) {
	if port == "" {
		log.Fatalln("Port is undefined, please setup your port")
	}

	srv := primeskillsserver.NewPrimeskillsServer()
	srv.SetException(exception.ErrorHandler)
	srv.SetStatusMethodNotAllowed(func(c *gin.Context) {
		panic(exception.NewMethodNotAllowedError(fmt.Sprintf("Path %s with methode is not allowed", c.Request.RequestURI)))
	})
	srv.SetStatusNotFound(func(c *gin.Context) {
		panic(exception.NewNotFoundErrorError(fmt.Sprintf("Path %s not found", c.Request.RequestURI)))
	})
	srv.RunServer(port, resource)
}
