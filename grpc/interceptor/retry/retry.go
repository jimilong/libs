// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

package retry

import (
	"fmt"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	AttemptMetadataKey = "x-retry-attempty"
)

// UnaryClientInterceptor returns a new retrying unary client interceptor.
//
// The default configuration of the interceptor is to not retry *at all*. This behaviour can be
// changed through options (e.g. WithMax) on creation of the interceptor or on call (through grpc.CallOptions).
func UnaryClientInterceptor(optFuncs ...CallOption) grpc.UnaryClientInterceptor {
	intOpts := reuseOrNewWithCallOptions(defaultOptions, optFuncs)
	return func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		grpcOpts, retryOpts := filterCallOptions(opts)
		callOpts := reuseOrNewWithCallOptions(intOpts, retryOpts)
		// short circuit for simplicity, and avoiding allocations.
		if callOpts.max == 0 {
			return invoker(parentCtx, method, req, reply, cc, grpcOpts...)
		}
		var lastErr error
		for attempt := uint(0); attempt < callOpts.max; attempt++ {
			if err := waitRetryBackoff(attempt, parentCtx, callOpts); err != nil {
				return err
			}
			callCtx := perCallContext(parentCtx, callOpts, attempt)
			lastErr = invoker(callCtx, method, req, reply, cc, grpcOpts...)
			// TODO(mwitkow): Maybe dial and transport errors should be retriable?
			if lastErr == nil {
				return nil
			}
			logTrace(parentCtx, "grpc_retry attempt: %d, got err: %v", attempt, lastErr)
			if isContextError(lastErr) {
				if parentCtx.Err() != nil {
					logTrace(parentCtx, "grpc_retry attempt: %d, parent context error: %v", attempt, parentCtx.Err())
					// its the parent context deadline or cancellation.
					return lastErr
				} else {
					logTrace(parentCtx, "grpc_retry attempt: %d, context error from retry call", attempt)
					// its the callCtx deadline or cancellation, in which case try again.
					continue
				}
			}
			if !isRetriable(lastErr, callOpts) {
				return lastErr
			}
		}
		return lastErr
	}
}

func waitRetryBackoff(attempt uint, parentCtx context.Context, callOpts *options) error {
	var waitTime time.Duration = 0
	if attempt > 0 {
		waitTime = callOpts.backoffFunc(attempt)
	}
	if waitTime > 0 {
		logTrace(parentCtx, "grpc_retry attempt: %d, backoff for %v", attempt, waitTime)
		timer := time.NewTimer(waitTime)
		select {
		case <-parentCtx.Done():
			timer.Stop()
			return contextErrToGrpcErr(parentCtx.Err())
		case <-timer.C:
		}
	}
	return nil
}

func isRetriable(err error, callOpts *options) bool {
	errCode := grpc.Code(err)
	if isContextError(err) {
		// context errors are not retriable based on user settings.
		return false
	}
	for _, code := range callOpts.codes {
		if code == errCode {
			return true
		}
	}
	return false
}

func isContextError(err error) bool {
	return grpc.Code(err) == codes.DeadlineExceeded || grpc.Code(err) == codes.Canceled
}

func perCallContext(parentCtx context.Context, callOpts *options, attempt uint) context.Context {
	ctx := parentCtx
	if callOpts.perCallTimeout != 0 {
		ctx, _ = context.WithTimeout(ctx, callOpts.perCallTimeout)
	}
	if attempt > 0 && callOpts.includeHeader {
		mdClone := metautils.ExtractOutgoing(ctx).Clone().Set(AttemptMetadataKey, fmt.Sprintf("%d", attempt))
		ctx = mdClone.ToOutgoing(ctx)
	}
	return ctx
}

func contextErrToGrpcErr(err error) error {
	switch err {
	case context.DeadlineExceeded:
		return grpc.Errorf(codes.DeadlineExceeded, err.Error())
	case context.Canceled:
		return grpc.Errorf(codes.Canceled, err.Error())
	default:
		return grpc.Errorf(codes.Unknown, err.Error())
	}
}

func logTrace(ctx context.Context, format string, a ...interface{}) {
	tr, ok := trace.FromContext(ctx)
	if !ok {
		return
	}
	tr.LazyPrintf(format, a...)
}
