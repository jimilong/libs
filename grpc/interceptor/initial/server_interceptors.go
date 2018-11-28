package initial

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerInterceptor returns a new unary server interceptors that initial request context.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			traceId := md.Get(CtxRequestIDKey)
			fmt.Println("inital: ", traceId)
			if len(traceId) > 0 {
				ctx = context.WithValue(ctx, CtxRequestIDKey, traceId[0])
			}
		}
		reply, err := handler(ctx, req)
		return reply, err
	}
}
