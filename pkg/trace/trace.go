package trace

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const Header = "X-Trace-ID"

type ctxKey struct{}

func NewID() string {
	return uuid.NewString()
}

func NewContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, traceID)
}

func FromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if val, ok := ctx.Value(ctxKey{}).(string); ok {
		return val, true
	}
	return "", false
}

func FieldFromContext(ctx context.Context) zap.Field {
	if traceID, ok := FromContext(ctx); ok {
		return zap.String("trace_id", traceID)
	}
	return zap.String("trace_id", "none")
}
