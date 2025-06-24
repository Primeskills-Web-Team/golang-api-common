package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleNotFound is a middleware function that handles 404 Not Found errors in a Gin application.
func HandleNotFound(ctx *gin.Context) {
	ctx.JSON(http.StatusNotFound, ResponseDto{
		Success: false,
		Message: "Route not found",
		Errors:  nil,
		Data:    nil,
	})
}
