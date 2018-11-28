package ratelimit

import (
	"context"

	"github.com/juju/ratelimit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a new unary server interceptors that add ratelimit
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	opt := evaluateOpt(opts)
	bucket := ratelimit.NewBucket(opt.fillInterval, opt.capacity)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if bucket.TakeAvailable(1) == 0 {
			return nil, status.Errorf(codes.ResourceExhausted, "request beyond rate limit")
		}

		return handler(ctx, req)
	}
}
