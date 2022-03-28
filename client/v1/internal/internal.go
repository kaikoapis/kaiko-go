package internal

import "google.golang.org/grpc"

type Config struct {
	APIKey string
	GRPCDialOptions []grpc.DialOption
	ICEEnabled bool
}
