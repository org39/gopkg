package grpc

import (
	"fmt"
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	grpcsdk "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server is a object that holds the state of the server
type Server struct {
	*grpcsdk.Server

	// Listener is the listener to use
	Listener net.Listener
}

// ServerOptions holds the options for the server
type ServerOptions struct {
	// Port is the port to listen on
	Port string

	// // MaxConnectionIdle is the maximum time a connection can be idle
	// MaxConnectionIdle time.Duration
	// // MaxConnectionAge is the maximum time a connection can be alive
	// MaxConnectionAge time.Duration
	// // MaxConnectionAgeGrace is the maximum time a connection can be alive after it was gracefully closed
	// MaxConnectionAgeGrace time.Duration
	// // PingTime is the time between server sent pings
	// PingTime time.Duration
	// // PingTimeout is the maximum time a client can take to respond to a server sent pings
	// PingTimeout time.Duration
}

// NewServer creates a new server
func NewServer(serverName string, options *ServerOptions) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", options.Port))
	if err != nil {
		return nil, err
	}

	// grpc server options
	opts := make([]grpcsdk.ServerOption, 0)

	// // keep alive options
	// opts = append(opts, grpcsdk.KeepaliveParams(keepalive.ServerParameters{
	// 	MaxConnectionIdle:     options.MaxConnectionIdle,
	// 	MaxConnectionAge:      options.MaxConnectionAge,
	// 	MaxConnectionAgeGrace: options.MaxConnectionAgeGrace,
	// 	Time:                  options.PingTime,
	// 	Timeout:               options.PingTimeout,
	// }))

	// empty interceptor chain for placehold
	opts = append(opts, grpcsdk.StreamInterceptor(grpc_middleware.ChainStreamServer()))
	opts = append(opts, grpcsdk.UnaryInterceptor(grpc_middleware.ChainUnaryServer()))

	// trace interceptor
	opts = append(opts, grpcsdk.ChainStreamInterceptor(otelgrpc.StreamServerInterceptor()))
	opts = append(opts, grpcsdk.ChainUnaryInterceptor(otelgrpc.UnaryServerInterceptor()))

	// service logger intercepter
	opts = append(opts, grpcsdk.ChainStreamInterceptor(streamServerLogInterceptor()))
	opts = append(opts, grpcsdk.ChainUnaryInterceptor(unaryServerLogInterceptor()))

	// create grpc server
	grpcServer := grpcsdk.NewServer(opts...)
	s := &Server{
		Server:   grpcServer,
		Listener: listener,
	}
	return s, nil
}

// NewDefaultServerOptions returns a new set of default options
func NewDefaultServerOptions() *ServerOptions {
	return &ServerOptions{
		Port: "7777",
		// MaxConnectionIdle:     time.Minute,
		// MaxConnectionAge:      time.Minute,
		// MaxConnectionAgeGrace: time.Minute,
		// PingTime:              time.Second,
		// PingTimeout:           time.Second,
	}
}

// Start starts the server
func (s *Server) Start() error {
	reflection.Register(s.Server)
	return s.Serve(s.Listener)
}

// Stop stops the server
func (s *Server) Stop() error {
	s.Server.GracefulStop()
	return nil
}
