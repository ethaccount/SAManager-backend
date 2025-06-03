package handler

import (
	"context"
	"reflect"

	"github.com/ethaccount/backend/src/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/shopspring/decimal"
)

func RegisterRoutes(ctx context.Context, router *gin.Engine, app *service.Application) {

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
			if value, ok := field.Interface().(decimal.Decimal); ok {
				return value.String()
			}
			return nil
		}, decimal.Decimal{})
	}

	SetMiddlewares(ctx, router)

	router.GET("/health", handleHealthCheck)

	passkeyHandler := NewPasskeyHandler(app.PasskeyService)
	jobHandler := NewJobHandler(app.JobService)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/register/begin", passkeyHandler.RegisterBegin())
		// v1.POST("/register/verify", passkeyHandler.RegisterVerify)
		// v1.POST("/login/options", passkeyHandler.LoginOptions)
		// v1.POST("/login/verify", passkeyHandler.LoginVerify)

		// Job management endpoints
		v1.POST("/jobs", jobHandler.RegisterJob)
	}

}
