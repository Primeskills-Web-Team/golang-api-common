package middleware

import "github.com/gin-gonic/gin"

// ResourceHeader is a middleware that sets the "X-Resource" header in the response.
func ResourceHeader(resourceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Resource", resourceName)
		c.Next()
	}
}
