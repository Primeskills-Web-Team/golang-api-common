package exception

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/response"
	"github.com/gin-gonic/gin"
)

type InternalServerError struct {
	Error string `json:"error"`
}

func NewInternalServerError(error string) InternalServerError {
	return InternalServerError{Error: error}
}

func internalServerErrorHandler(ctx *gin.Context, err interface{}, resource string) bool {
	exception, ok := err.(InternalServerError)
	if ok {
		ctx.JSON(response.HttpCode[response.GeneralError], response.ConstructResponse(
			response.GeneralError,
			response.Descriptions[response.GeneralError],
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
