package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/org39/gopkg/log"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

const (
	toMilli = 1e6
)

func loggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()
			ctx := req.Context()

			duration := time.Since(start)
			status := res.Status

			traceLogger := log.LoggerWithSpan(ctx)
			accessLog := traceLogger.WithFields(logrus.Fields{
				"http_status":        status,
				"http_host":          req.Host,
				"http_latency":       float64(duration) / float64(toMilli),
				"http_latency_human": duration.String(),
				"http_method":        req.Method,
				"http_uri":           req.RequestURI,
				"http_remote_ip":     c.RealIP(),
			})

			msg := fmt.Sprintf("%s %s", req.Method, req.RequestURI)
			switch {
			case status >= http.StatusInternalServerError:
				accessLog.Error(msg)
			case status >= http.StatusBadRequest:
				accessLog.Warn(msg)
			case status >= http.StatusMultipleChoices:
				accessLog.Debug(msg)
			default:
				accessLog.Debug(msg)
			}

			return nil
		}
	}
}
