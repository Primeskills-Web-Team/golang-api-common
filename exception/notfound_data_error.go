package exception

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/response"
	"github.com/gin-gonic/gin"
)

type NotFoundDataErrorError struct {
	Error string `json:"error"`
}

func NewNotFoundDataErrorError(error string) NotFoundDataErrorError {
	return NotFoundDataErrorError{Error: error}
}

func notFoundDataErrorErrorHandler(ctx *gin.Context, err interface{}, resource string) bool {
	exception, ok := err.(NotFoundDataErrorError)
	if ok {
		ctx.JSON(response.HttpCode[response.DataNotFound], response.ConstructResponse(
			response.DataNotFound,
			response.Descriptions[response.DataNotFound],
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
