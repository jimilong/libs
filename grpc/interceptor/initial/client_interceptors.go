package initial

import (
	"context"

	"github.com/satori/go.uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	CtxRequestIDKey = "X-Request-ID"
)

// UnaryClientInterceptor returns a new unary client interceptor that initial request context
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		traceId, ok := ctx.Value(CtxRequestIDKey).(string)
		if !ok {
			if u, err := uuid.NewV4(); err == nil {
				traceId = u.String()
			}
		}
		ctx = metadata.AppendToOutgoingContext(ctx, CtxRequestIDKey, traceId)
		ctx = context.WithValue(ctx, CtxRequestIDKey, traceId)

		err := invoker(ctx, method, req, reply, cc, opts...)
		return err
	}
}
