package dashboardhttp

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

const (
	defaultStackSize = 4096
)

// RecoverMiddleware recovers from panics and logs them.
func RecoverMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context) {
				if err := recover(); err != nil {
					logger := zerolog.Ctx(ctx)

					// Get stack trace
					stack := make([]byte, defaultStackSize)
					length := runtime.Stack(stack, false)
					stackTrace := string(stack[:length])

					// Log the panic
					logger.Error().
						Interface("panic", err).
						Str("stack", stackTrace).
						Str("method", r.Method).
						Str("url", r.URL.String()).
						Str("remote_addr", r.RemoteAddr).
						Msg("HTTP handler panic recovered")

					// Send error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = fmt.Fprintf(w, `{"error":"internal server error","message":"request failed due to internal error"}`)
				}
			}(r.Context())

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware logs HTTP requests.
func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger := zerolog.Ctx(ctx)
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log request
			duration := time.Since(start)
			logEvent := logger.Info().
				Str("method", r.Method).
				Str("url", r.URL.String()).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Int("status", wrapped.statusCode).
				Dur("duration", duration)

			// Add referer if present
			if referer := r.Header.Get("Referer"); referer != "" {
				logEvent = logEvent.Str("referer", referer)
			}

			// Add content length if present
			if contentLength := r.Header.Get("Content-Length"); contentLength != "" {
				logEvent = logEvent.Str("content_length", contentLength)
			}

			logEvent.Msg("HTTP request")
		})
	}
}

// CORSMiddleware adds CORS headers.
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware adds security headers.
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss:")

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware implements rate limiting.
func RateLimitMiddleware(requestsPerSecond int, burst int) func(http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow API endpoints without rate limiting for admin access
			if strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)

				return
			}

			// Rate limit static files and UI
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = fmt.Fprintf(w, `{"error":"rate limit exceeded","message":"too many requests"}`)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware adds request timeout.
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Update request with new context
			r = r.WithContext(ctx)

			// Create channel for handling timeout
			done := make(chan struct{})

			go func() {
				defer close(done)

				next.ServeHTTP(w, r)
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Request completed successfully
			case <-ctx.Done():
				// Timeout occurred
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestTimeout)
				_, _ = fmt.Fprintf(w, `{"error":"request timeout","message":"request took too long to process"}`)
			}
		})
	}
}

// HealthCheckMiddleware provides basic health check endpoint.
func HealthCheckMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter

	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ChainMiddleware chains multiple middleware functions.
func ChainMiddleware(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}

		return next
	}
}
