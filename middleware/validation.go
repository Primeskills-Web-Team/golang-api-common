package middleware

import (
	"net/http"

	"github.com/Primeskills-Web-Team/golang-api-common/v2/common"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ValidationMiddleware is an interface that defines methods for validating request data
type ValidationMiddleware interface {
	ValidateBody(ctx *gin.Context, v any)
	ValidateQuery(ctx *gin.Context, v any)
}

// validationMiddleware is a struct that implements the ValidationMiddleware interface
type validationMiddleware struct {
	validator *validator.Validator
}

// ValidateBody validates the request body against the provided struct
func (vm *validationMiddleware) ValidateBody(ctx *gin.Context, v any) {
	vm.validate(ctx, v, ctx.Bind)
}

// ValidateQuery validates the request query parameters against the provided struct
func (vm *validationMiddleware) ValidateQuery(ctx *gin.Context, v any) {
	vm.validate(ctx, v, ctx.BindQuery)
}

// validate is a helper function that performs the actual validation logic
func (vm *validationMiddleware) validate(ctx *gin.Context, v any, bindFunc func(any) error) {
	if err := bindFunc(v); err != nil {
		log.Error().Err(err).Msg("Failed to bind request")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, common.ResponseDto{
			Success: false,
			Message: "Invalid request",
			Errors:  nil,
			Data:    nil,
		})
		return
	}

	if validationErrors := vm.validator.ValidateStruct(v); validationErrors != nil {
		ctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, common.ResponseDto{
			Success: false,
			Message: "Validation error",
			Errors:  validationErrors,
			Data:    nil,
		})
		return
	}

	ctx.Set("validatedData", v)
	ctx.Next()
}

// NewValidationMiddleware creates a new instance of ValidationMiddleware
func NewValidationMiddleware(v *validator.Validator) ValidationMiddleware {
	return &validationMiddleware{
		validator: v,
	}
}
