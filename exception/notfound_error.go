package exception

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/response"
	"github.com/gin-gonic/gin"
)

type NotFoundErrorError struct {
	Error string `json:"error"`
}

func NewNotFoundErrorError(error string) NotFoundErrorError {
	return NotFoundErrorError{Error: error}
}

func notFoundErrorHandler(ctx *gin.Context, err interface{}, resource string) bool {
	exception, ok := err.(NotFoundErrorError)
	if ok {
		ctx.JSON(response.HttpCode[response.RouteNotFound], response.ConstructResponse(
			response.RouteNotFound,
			response.Descriptions[response.RouteNotFound],
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
