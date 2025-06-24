package middleware

import (
	"net/http"

	"github.com/Primeskills-Web-Team/golang-api-common/v2/common"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Recover is a middleware that recovers from panics and writes a 500 if there was one.
func Recover(ctx *gin.Context, recovered any) {
	log.Error().Msgf("Panic: %v", recovered)
	ctx.AbortWithStatusJSON(http.StatusInternalServerError, common.ResponseDto{
		Success: false,
		Message: "Something went wrong",
		Errors:  nil,
		Data:    nil,
	})
}
