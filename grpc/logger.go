package grpc

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/org39/gopkg/log"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/sirupsen/logrus"
	grpcsdk "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	toMilli = 1e6
)

func streamServerLogInterceptor() grpcsdk.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpcsdk.ServerStream,
		info *grpcsdk.StreamServerInfo,
		handler grpcsdk.StreamHandler,
	) error {
		ctx := stream.Context()
		startTime := time.Now()
		deadline, deadlineIsSet := ctx.Deadline()

		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = ctx

		err := handler(srv, wrapped)

		code := status.Code(err)
		service := path.Dir(info.FullMethod)[1:]
		method := path.Base(info.FullMethod)
		duration := time.Since(startTime)

		// skip logging for grpc.health.v1.Health service
		if service == "grpc.health.v1.Health" {
			return err
		}

		traceLogger := log.LoggerWithSpan(ctx)
		accessLog := traceLogger.WithFields(logrus.Fields{
			"grpc_method":        method,
			"grpc_service":       service,
			"grpc_code":          code,
			"grpc_code_human":    code.String(),
			"grpc_latency":       float64(duration) / float64(toMilli),
			"grpc_latency_human": duration.String(),
		})
		if deadlineIsSet {
			accessLog = accessLog.WithField("grpc_deadline", deadline)
		}
		if err != nil {
			accessLog = accessLog.WithError(err)
		}

		msg := fmt.Sprintf("%s %s", service, method)
		sendLog(err, code, accessLog, msg)
		return err
	}
}

func unaryServerLogInterceptor() grpcsdk.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpcsdk.UnaryServerInfo,
		handler grpcsdk.UnaryHandler,
	) (interface{}, error) {
		startTime := time.Now()
		deadline, deadlineIsSet := ctx.Deadline()

		resp, err := handler(ctx, req)

		code := status.Code(err)
		service := path.Dir(info.FullMethod)[1:]
		method := path.Base(info.FullMethod)
		duration := time.Since(startTime)

		// skip logging for grpc.health.v1.Health service
		if service == "grpc.health.v1.Health" {
			return resp, err
		}

		traceLogger := log.LoggerWithSpan(ctx)
		accessLog := traceLogger.WithFields(logrus.Fields{
			"grpc_method":        method,
			"grpc_service":       service,
			"grpc_code":          code,
			"grpc_code_human":    code.String(),
			"grpc_latency":       float64(duration) / float64(toMilli),
			"grpc_latency_human": duration.String(),
		})
		if deadlineIsSet {
			accessLog = accessLog.WithField("grpc_deadline", deadline)
		}
		if err != nil {
			accessLog = accessLog.WithError(err)
		}

		msg := fmt.Sprintf("%s %s", service, method)
		sendLog(err, code, accessLog, msg)
		return resp, err
	}
}

func sendLog(err error, code codes.Code, accessLog *logrus.Entry, msg string) {
	if err != nil {
		accessLog.Error(msg)
		return
	}

	switch code {
	case codes.OK:
		accessLog.Debug(msg)
	case codes.Canceled:
		accessLog.Debug(msg)
	case codes.Unknown:
		accessLog.Warning(msg)
	case codes.InvalidArgument:
		accessLog.Warning(msg)
	case codes.DeadlineExceeded:
		accessLog.Warning(msg)
	case codes.NotFound:
		accessLog.Warning(msg)
	case codes.AlreadyExists:
		accessLog.Warning(msg)
	case codes.PermissionDenied:
		accessLog.Warning(msg)
	case codes.Unauthenticated:
		accessLog.Warning(msg)
	case codes.ResourceExhausted:
		accessLog.Warning(msg)
	case codes.FailedPrecondition:
		accessLog.Warning(msg)
	case codes.Aborted:
		accessLog.Warning(msg)
	case codes.OutOfRange:
		accessLog.Warning(msg)
	case codes.Unimplemented:
		accessLog.Error(msg)
	case codes.Internal:
		accessLog.Error(msg)
	case codes.Unavailable:
		accessLog.Error(msg)
	case codes.DataLoss:
		accessLog.Error(msg)
	default:
		accessLog.Error(msg)
	}
}
