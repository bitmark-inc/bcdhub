package helpers

import (
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// InitSentry -
func InitSentry(env, dsn string, debug bool) {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Debug:            debug,
		AttachStacktrace: true,
	}); err != nil {
		log.Printf("Sentry initialization failed: %v\n", err)
	}
}

// SentryMiddleware -
func SentryMiddleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{})
}

// SetTagSentry -
func SetTagSentry(key, value string) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag(key, value)
	})
}

// SetUserIDSentry -
func SetUserIDSentry(id string) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID: id,
		})
	})
}

// CatchPanicSentry -
func CatchPanicSentry() {
	if err := recover(); err != nil {
		sentry.CurrentHub().Recover(err)
		sentry.Flush(time.Second * 5)
	}
}

// CatchErrorSentry -
func CatchErrorSentry(err error) {
	sentry.CaptureException(err)
	sentry.Flush(time.Second * 5)
}

// GetLocalSentry -
func GetLocalSentry() *sentry.Hub {
	return sentry.CurrentHub().Clone()
}

// SetLocalTagSentry -
func SetLocalTagSentry(hub *sentry.Hub, key, value string) {
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag(key, value)
	})
}

// LocalCatchErrorSentry -
func LocalCatchErrorSentry(hub *sentry.Hub, err error) {
	hub.CaptureException(err)
	hub.Flush(time.Second * 5)
}

// LocalCatchPanicSentry -
func LocalCatchPanicSentry(hub *sentry.Hub) {
	if err := recover(); err != nil {
		hub.CaptureMessage(err.(string))
		hub.Flush(time.Second * 5)
	}
}
