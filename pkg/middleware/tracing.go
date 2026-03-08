package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"admin-service/pkg/trace"
)

// injects and propagates trace IDs while logging every request
func TraceMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// create and inject trace id if not found in header
		traceID := c.GetHeader(trace.Header)
		if traceID == "" {
			traceID = trace.NewID()
		}

		c.Writer.Header().Set(trace.Header, traceID)
		ctx := trace.NewContext(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()
		c.Next()

		if logger != nil {
			fields := []zap.Field{
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("client_ip", c.ClientIP()),
				zap.Int("status", c.Writer.Status()),
				zap.Int64("elapsed_ms", time.Since(start).Milliseconds()),
				trace.FieldFromContext(ctx),
			}
			logger.Info("http request", fields...)
		}
	}
}
