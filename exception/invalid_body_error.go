package exception

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/response"
	"github.com/gin-gonic/gin"
)

type BadRequestBodyError struct {
	Error string `json:"error"`
}

func NewBadRequestBodyError(error string) BadRequestBodyError {
	return BadRequestBodyError{Error: error}
}

func badRequestBodyErrorHandler(ctx *gin.Context, err interface{}, resource string) bool {
	exception, ok := err.(BadRequestBodyError)
	if ok {
		ctx.JSON(response.HttpCode[response.FailedValidationForm], response.ConstructResponse(
			response.FailedValidationForm,
			response.Descriptions[response.FailedValidationForm],
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
