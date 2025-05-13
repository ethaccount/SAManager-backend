package handler

import (
	"context"

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
