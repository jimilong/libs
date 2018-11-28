package ratelimit

import (
	"time"
)

const (
	DefaultBucketFillInterval = time.Millisecond
	DefaultBucketCapacity     = 1000
)

var (
	defaultOptions = &options{
		fillInterval: DefaultBucketFillInterval,
		capacity:     DefaultBucketCapacity,
	}
)

// WithFillInterval set the bucket fill interval.
func WithFillInterval(fillInterval time.Duration) Option {
	return func(o *options) {
		o.fillInterval = fillInterval
	}
}

// WithCapacity set the bucket capacity.
func WithCapacity(capacity int64) Option {
	return func(o *options) {
		o.capacity = capacity
	}
}

type options struct {
	fillInterval time.Duration
	capacity     int64
}

// Option is a function for setting the logging option.
type Option func(*options)

func evaluateOpt(opts []Option) *options {
	optCopy := &options{}
	*optCopy = *defaultOptions
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}
