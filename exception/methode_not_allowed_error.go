package exception

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/response"
	"github.com/gin-gonic/gin"
)

type MethodNotAllowedError struct {
	Error string `json:"error"`
	Ctx   *gin.Context
}

func NewMethodNotAllowedError(error string) MethodNotAllowedError {
	return MethodNotAllowedError{Error: error}
}

func methodNotAllowedHandler(ctx *gin.Context, err interface{}, resource string) bool {
	exception, ok := err.(MethodNotAllowedError)
	if ok {
		ctx.JSON(405, response.ConstructResponse(
			"045",
			"Status Methode Not Allowed",
			fmt.Sprintf("%v", exception.Error),
			resource,
			false,
			nil,
		))
		return true
	} else {
		return false
	}
}
