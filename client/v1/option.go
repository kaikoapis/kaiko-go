package clientv1

import (
	"google.golang.org/grpc"

	"kaiko.com/go/client/v1/internal"
)

// Option is a client option.
type Option func(*internal.Config)

// WithAPIKey sets API key to ak.
func WithAPIKey(ak string) Option {
	return func(c *internal.Config) {
		c.APIKey = ak
	}
}

// WithICEEnabled enables ICE connectivity.
func WithICEEnabled() Option {
	return func(c *internal.Config) {
		c.ICEEnabled = true
	}
}

// WithGRPCDialOptions adds opts to the list of gRPC dial options.
func WithGRPCDialOptions(opts ...grpc.DialOption) Option {
	return func(c *internal.Config) {
		c.GRPCDialOptions = opts
	}
}

func newConfig(opts ...Option) *internal.Config {
	var cfg internal.Config

	for _, opt := range opts {
		opt(&cfg)
	}

	return &cfg
}
