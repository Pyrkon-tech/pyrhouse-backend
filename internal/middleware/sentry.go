package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// maxCapturedBody caps how much of a 5xx response body is copied for reporting.
const maxCapturedBody = 4096

// flushTimeout is how long a flush waits for pending events to reach Sentry.
const flushTimeout = 2 * time.Second

// SentryOptions configures error reporting.
type SentryOptions struct {
	DSN              string
	Environment      string
	Release          string
	TracesSampleRate float64
}

var sentryEnabled bool

// InitSentry starts error reporting. An empty DSN is not an error: reporting
// stays off and every helper below becomes a no-op, so the app runs unchanged
// locally and in tests.
func InitSentry(opts SentryOptions) error {
	if opts.DSN == "" {
		log.Println("[Sentry]: SENTRY_DSN not set - error reporting disabled")
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              opts.DSN,
		Environment:      opts.Environment,
		Release:          opts.Release,
		AttachStacktrace: true,
		EnableTracing:    opts.TracesSampleRate > 0,
		TracesSampleRate: opts.TracesSampleRate,
		BeforeSend:       scrubEvent,
	})
	if err != nil {
		return err
	}

	sentryEnabled = true
	log.Printf("[Sentry]: Initialized (environment: %s, release: %s, traces sample rate: %.2f)",
		opts.Environment, opts.Release, opts.TracesSampleRate)

	return nil
}

// SentryEnabled reports whether error reporting is active.
func SentryEnabled() bool {
	return sentryEnabled
}

// FlushSentry waits for queued events to be delivered. Call it before exiting.
func FlushSentry() {
	if sentryEnabled {
		sentry.Flush(flushTimeout)
	}
}

// CaptureFatal reports an error that is about to terminate the process. It
// flushes synchronously, because the caller will not come back.
func CaptureFatal(msg string) {
	if !sentryEnabled {
		return
	}

	hub := sentry.CurrentHub()
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelFatal)
		hub.CaptureMessage(msg)
	})

	sentry.Flush(flushTimeout)
}

// SentryHandlers returns the middlewares that feed Sentry, in the order they
// must be registered. It returns nil when reporting is disabled.
func SentryHandlers() []gin.HandlerFunc {
	if !sentryEnabled {
		return nil
	}

	return []gin.HandlerFunc{
		// Repanic hands the panic back so RecoveryMiddleware, registered
		// earlier and therefore outside this one, still returns the standard
		// JSON 500 to the client.
		sentrygin.New(sentrygin.Options{Repanic: true}),
		sentryReporter(),
	}
}

// sentryReporter reports 5xx responses. Handlers in this codebase answer with
// c.JSON(500, gin.H{"error": ..., "details": ...}) rather than returning an
// error upwards, so the response body is the only description of the failure.
//
// Panics are not reported here: the code after c.Next() does not run while a
// panic unwinds, which leaves them to sentrygin alone and avoids duplicates.
func sentryReporter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isHealthPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		writer := &errorBodyWriter{ResponseWriter: c.Writer}
		c.Writer = writer

		c.Next()

		status := c.Writer.Status()
		if status < 500 {
			return
		}

		hub := sentrygin.GetHubFromContext(c)
		if hub == nil {
			return
		}

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		title, details := parseErrorBody(writer.body.Bytes())

		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetLevel(sentry.LevelError)
			scope.SetTag("route", route)
			scope.SetTag("status", strconv.Itoa(status))
			if details != "" {
				scope.SetExtra("details", details)
			}
			// Group by route and error title. The details usually carry ids and
			// would otherwise open a fresh issue for every single request.
			scope.SetFingerprint([]string{c.Request.Method, route, title})

			hub.CaptureMessage(fmt.Sprintf("%d %s %s: %s", status, c.Request.Method, route, title))
		})
	}
}

// errorBodyWriter keeps a copy of the response body, but only for error
// responses - buffering every payload would cost memory on large listings.
type errorBodyWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *errorBodyWriter) Write(b []byte) (int, error) {
	w.capture(b)
	return w.ResponseWriter.Write(b)
}

func (w *errorBodyWriter) WriteString(s string) (int, error) {
	w.capture([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

func (w *errorBodyWriter) capture(b []byte) {
	if w.Status() < 500 {
		return
	}

	remaining := maxCapturedBody - w.body.Len()
	if remaining <= 0 {
		return
	}
	if len(b) > remaining {
		b = b[:remaining]
	}

	w.body.Write(b)
}

// parseErrorBody pulls the title and the detail out of the standard error
// response shape, falling back to the raw body for anything else.
func parseErrorBody(body []byte) (title, details string) {
	if len(body) == 0 {
		return "no response body", ""
	}

	var parsed struct {
		Error   string `json:"error"`
		Details string `json:"details"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		return strings.TrimSpace(string(body)), ""
	}

	title = parsed.Error
	if title == "" {
		title = parsed.Message
	}
	if title == "" {
		return strings.TrimSpace(string(body)), ""
	}

	details = parsed.Details
	if details == "" {
		details = parsed.Message
	}

	return title, details
}

// scrubEvent removes credentials before anything leaves the process.
func scrubEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	if event.Request != nil {
		for _, header := range []string{"Authorization", "Cookie", "X-Api-Key"} {
			delete(event.Request.Headers, header)
		}
		event.Request.Cookies = ""
	}

	return event
}

func isHealthPath(path string) bool {
	return strings.HasPrefix(path, "/health")
}
