package exception

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/response"
	"github.com/gin-gonic/gin"
)

type UnAuthorizeError struct {
	Error string `json:"error"`
}

func NewUnAuthorizeError(error string) UnAuthorizeError {
	return UnAuthorizeError{Error: error}
}

func unAuthorizeErrorHandler(ctx *gin.Context, err interface{}, resource string) bool {
	exception, ok := err.(UnAuthorizeError)
	if ok {
		ctx.JSON(response.HttpCode[response.FailedAuthorization], response.ConstructResponse(
			response.FailedAuthorization,
			response.Descriptions[response.FailedAuthorization],
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
