package middleware

import (
	"errors"
	"net/http"

	"github.com/Primeskills-Web-Team/golang-api-common/v2/common"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/exception"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Exception is a middleware that handles exceptions in the application.
func Exception() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			var httpException *exception.HTTPException
			if errors.As(err.Err, &httpException) {
				c.AbortWithStatusJSON(httpException.StatusCode, common.ResponseDto{
					Success: false,
					Message: httpException.Error(),
					Errors:  nil,
					Data:    nil,
				})
				return
			}

			c.AbortWithStatusJSON(http.StatusInternalServerError, common.ResponseDto{
				Success: false,
				Message: "Something went wrong",
				Errors:  nil,
				Data:    nil,
			})

			log.Error().Msgf("Error: %v", err.Err)
			return
		}
	}
}
