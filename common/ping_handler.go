package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandlePing is a simple handler that responds with a JSON object indicating the service is alive.
func HandlePing(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResponseDto{
		Success: true,
		Message: "pong",
		Errors:  nil,
		Data:    nil,
	})
}
