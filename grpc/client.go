package grpc

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	grpcsdk "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClient creates a new client
func NewClient(address string) (*grpcsdk.ClientConn, error) {
	conn, err := grpcsdk.Dial(address, grpcsdk.WithTransportCredentials(insecure.NewCredentials()),
		grpcsdk.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpcsdk.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
