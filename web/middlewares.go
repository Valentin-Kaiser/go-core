package web

import (
	"net/http"

	"github.com/Valentin-Kaiser/go-essentials/flag"
	"github.com/Valentin-Kaiser/go-essentials/version"
	"github.com/rs/zerolog/log"
)

var (
	securityHeaders = map[string]string{
		"ETag":                      version.GitCommit,
		"Cache-Control":             "public, must-revalidate, max-age=86400",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Referrer-Policy":           "no-referrer-when-downgrade",
	}
	corsHeaders = map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization, X-Real-IP",
	}
)

type Middleware func(http.Handler) http.Handler

// securityHeaderMiddleware is a middleware that adds security headers to the response
// It is used to prevent attacks like XSS, clickjacking, etc.
func securityHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, value := range securityHeaders {
			w.Header().Set(key, value)
		}
		if flag.Debug {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		}
		next.ServeHTTP(w, r)
	})
}

// corsHeaderMiddleware is a middleware that adds CORS headers to the response
// It is used to allow cross-origin requests from the client
func corsHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		for key, value := range corsHeaders {
			w.Header().Set(key, value)
		}
		next.ServeHTTP(w, r)
	})
}

// logMiddleware is a middleware that logs the request and response
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &ResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		loglevel := log.Debug
		if rw.status >= 400 {
			loglevel = log.Warn
		}
		if rw.status >= 500 {
			loglevel = log.Error
		}

		loglevel().
			Str("remote", r.RemoteAddr).
			Str("real-ip", r.Header.Get("X-Real-IP")).
			Str("host", r.Host).
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Str("user-agent", r.UserAgent()).
			Str("referer", r.Referer()).
			Str("status", rw.Status()).
			Msg("request")
	})
}
