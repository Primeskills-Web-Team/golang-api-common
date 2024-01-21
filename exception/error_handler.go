package exception

import (
	"fmt"
	"github.com/Primeskills-Web-Team/golang-api-common/response"
	"github.com/gin-gonic/gin"
)

func ErrorHandler(ctx *gin.Context, err interface{}, resource string) {
	if internalServerErrorHandler(ctx, err, resource) {
		return
	}

	if unAuthorizeErrorHandler(ctx, err, resource) {
		return
	}

	if badRequestErrorHandler(ctx, err, resource) {
		return
	}

	if badRequestBodyErrorHandler(ctx, err, resource) {
		return
	}

	if notFoundDataErrorErrorHandler(ctx, err, resource) {
		return
	}

	if notFoundErrorHandler(ctx, err, resource) {
		return
	}

	if unAuthenticateHandle(ctx, err, resource) {
		return
	}

	ctx.JSON(response.HttpCode[response.GeneralError], response.ConstructResponse(
		response.GeneralError,
		response.Descriptions[response.GeneralError],
		fmt.Sprintf("%v", err),
		resource,
		false,
		nil,
	))
}
