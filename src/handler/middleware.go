package handler

import (
	"context"
	"errors"

	"github.com/ethaccount/backend/src/domain"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func SetMiddlewares(ctx context.Context, ginRouter *gin.Engine) {
	ginRouter.Use(LoggerMiddleware(ctx))
}

func LoggerMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {

		zlog := zerolog.Ctx(ctx).With().
			Str("path", c.FullPath()).
			Str("method", c.Request.Method).
			Logger()
		ctx = zlog.WithContext(ctx)
		c.Request = c.Request.WithContext(zlog.WithContext(ctx))
		c.Next()
	}
}

// SharedSecretMiddleware validates the X-API-Secret header
func SharedSecretMiddleware(apiSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the secret from request header
		providedSecret := c.GetHeader("X-API-Secret")

		// Check if secret is provided
		if providedSecret == "" {
			err := domain.NewError(
				domain.ErrorCodeAuthNotAuthenticated,
				errors.New("missing API secret header"),
				domain.WithMsg("Missing API secret"),
			)
			respondWithError(c, err)
			return
		}

		// Validate the secret
		if providedSecret != apiSecret {
			err := domain.NewError(
				domain.ErrorCodeAuthNotAuthenticated,
				errors.New("invalid API secret provided"),
				domain.WithMsg("Invalid API secret"),
			)
			respondWithError(c, err)
			return
		}

		// Secret is valid, proceed to next handler
		c.Next()
	}
}
