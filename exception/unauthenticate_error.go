package exception

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/response"
	"github.com/gin-gonic/gin"
)

type UnAuthenticateError struct {
	Error string `json:"error"`
}

func NewUnAuthenticate(error string) UnAuthenticateError {
	return UnAuthenticateError{Error: error}
}

func unAuthenticateHandle(ctx *gin.Context, err interface{}, resource string) bool {
	exception, ok := err.(UnAuthenticateError)
	if ok {
		ctx.JSON(response.HttpCode[response.FailedAuthenticate], response.ConstructResponse(
			response.FailedAuthenticate,
			response.Descriptions[response.FailedAuthenticate],
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
