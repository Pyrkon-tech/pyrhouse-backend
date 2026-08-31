package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ingestSpy stands in for Sentry: it collects the raw envelopes the SDK sends.
type ingestSpy struct {
	mu        sync.Mutex
	envelopes []string
	server    *httptest.Server
}

func newIngestSpy(t *testing.T) *ingestSpy {
	t.Helper()

	spy := &ingestSpy{}
	spy.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		spy.mu.Lock()
		spy.envelopes = append(spy.envelopes, string(body))
		spy.mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(spy.server.Close)

	return spy
}

func (s *ingestSpy) dsn() string {
	return "http://publickey@" + strings.TrimPrefix(s.server.URL, "http://") + "/1"
}

func (s *ingestSpy) collected() []string {
	FlushSentry()

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]string, len(s.envelopes))
	copy(out, s.envelopes)

	return out
}

func newTestRouter(t *testing.T, spy *ingestSpy) *gin.Engine {
	t.Helper()

	require.NoError(t, InitSentry(SentryOptions{
		DSN:         spy.dsn(),
		Environment: "test",
		Release:     "test-release",
	}))
	t.Cleanup(func() { sentryEnabled = false })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RecoveryMiddleware())

	handlers := SentryHandlers()
	require.Len(t, handlers, 2, "Sentry middlewares should be registered when a DSN is set")
	for _, handler := range handlers {
		router.Use(handler)
	}

	router.GET("/api/v1/boom", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Unable to get asset",
			"details": "sql: no rows in result set",
		})
	})
	router.GET("/api/v1/forbidden", func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
	})
	router.GET("/api/v1/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "fine"})
	})
	router.GET("/api/v1/panic", func(c *gin.Context) {
		panic("boom in handler")
	})
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "health blew up"})
	})

	return router
}

func TestSentryReporter(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		wantStatus     int
		wantEvents     int
		wantInEnvelope []string
	}{
		{
			name:       "5xx response is reported with error title and details",
			path:       "/api/v1/boom",
			wantStatus: http.StatusInternalServerError,
			wantEvents: 1,
			wantInEnvelope: []string{
				"500 GET /api/v1/boom: Unable to get asset",
				"sql: no rows in result set",
				"test-release",
			},
		},
		{
			name:       "4xx response is not reported",
			path:       "/api/v1/forbidden",
			wantStatus: http.StatusForbidden,
			wantEvents: 0,
		},
		{
			name:       "successful response is not reported",
			path:       "/api/v1/ok",
			wantStatus: http.StatusOK,
			wantEvents: 0,
		},
		{
			name:       "health endpoint is skipped even on 5xx",
			path:       "/health",
			wantStatus: http.StatusInternalServerError,
			wantEvents: 0,
		},
		{
			name:       "panic is reported once and still answers with the standard JSON 500",
			path:       "/api/v1/panic",
			wantStatus: http.StatusInternalServerError,
			wantEvents: 1,
			wantInEnvelope: []string{
				"boom in handler",
				"Internal Server Error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := newIngestSpy(t)
			router := newTestRouter(t, spy)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, recorder.Code)

			envelopes := spy.collected()
			assert.Len(t, envelopes, tt.wantEvents)

			for _, want := range tt.wantInEnvelope {
				if want == "Internal Server Error" {
					// the panic case: the client still gets the standard body
					assert.Contains(t, recorder.Body.String(), want)
					continue
				}
				require.NotEmpty(t, envelopes)
				assert.Contains(t, envelopes[0], want)
			}
		})
	}
}

func TestSentryDisabledWithoutDSN(t *testing.T) {
	require.NoError(t, InitSentry(SentryOptions{}))
	t.Cleanup(func() { sentryEnabled = false })

	assert.False(t, SentryEnabled())
	assert.Nil(t, SentryHandlers(), "no Sentry middleware should be registered without a DSN")

	// The helpers must stay callable and silent when reporting is off.
	CaptureFatal("this must not panic")
	FlushSentry()
}

func TestParseErrorBody(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantTitle   string
		wantDetails string
	}{
		{
			name:        "standard error response",
			body:        `{"error":"Unable to get asset","details":"sql: no rows"}`,
			wantTitle:   "Unable to get asset",
			wantDetails: "sql: no rows",
		},
		{
			name:        "recovery response shape",
			body:        `{"error":"Internal Server Error","message":"Aplikacja napotkala blad"}`,
			wantTitle:   "Internal Server Error",
			wantDetails: "Aplikacja napotkala blad",
		},
		{
			name:      "plain text body",
			body:      "upstream exploded",
			wantTitle: "upstream exploded",
		},
		{
			name:      "empty body",
			body:      "",
			wantTitle: "no response body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, details := parseErrorBody([]byte(tt.body))

			assert.Equal(t, tt.wantTitle, title)
			assert.Equal(t, tt.wantDetails, details)
		})
	}
}
