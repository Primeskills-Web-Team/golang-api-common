package exception

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/response"
	"github.com/gin-gonic/gin"
)

type BadRequestError struct {
	Error string `json:"error"`
	Ctx   *gin.Context
}

func NewBadRequestError(error string) BadRequestError {
	return BadRequestError{Error: error}
}

func badRequestErrorHandler(ctx *gin.Context, err interface{}, resource string) bool {
	exception, ok := err.(BadRequestError)
	if ok {
		ctx.JSON(400, response.ConstructResponse(
			"040",
			"Bad request",
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
