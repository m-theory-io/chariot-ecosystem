package logs

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
}

type ZapLogger struct {
	logger *zap.Logger
}

func NewZapLogger() *ZapLogger {
	logger, _ := zap.NewProduction()
	return &ZapLogger{logger: logger}
}

func (l *ZapLogger) Get() *zap.Logger {
	return l.logger
}

func (l *ZapLogger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

func (l *ZapLogger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

func (l *ZapLogger) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}

func (l *ZapLogger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

func ZapLoggerMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			logger.Info("incoming request",
				zap.String("method", req.Method),
				zap.String("uri", req.RequestURI),
				zap.String("remote_addr", req.RemoteAddr),
			)
			err := next(c)
			logger.Info("response sent",
				zap.Int("status", res.Status),
			)
			return err
		}
	}
}
