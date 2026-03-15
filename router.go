package main

import (
	"app/helpers"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"app/controllers"
	"app/models"
	"app/modules/mvpchat"
	"app/modules/profile"
	"app/observability"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestCounter      *prometheus.CounterVec
	metricsRegistry     *prometheus.Registry
	registerMetricsOnce sync.Once
)

func initMetrics() {
	registerMetricsOnce.Do(func() {
		metricsRegistry = prometheus.NewRegistry()

		requestCounter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests received labeled by route and status.",
			},
			[]string{"method", "route", "status_code"},
		)

		metricsRegistry.MustRegister(prometheus.NewGoCollector())
		metricsRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
		metricsRegistry.MustRegister(requestCounter)
	})
}

func (app *App) router() http.Handler {
	initMetrics()
	config := cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders: []string{"Link"},
		MaxAge:         300,
	}

	allowedOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if allowedOrigins != "" {
		config.AllowedOrigins = strings.Split(allowedOrigins, ",")
		for i := range config.AllowedOrigins {
			config.AllowedOrigins[i] = strings.TrimSpace(config.AllowedOrigins[i])
		}
	}
	config.AllowCredentials = len(config.AllowedOrigins) > 0

	mux := chi.NewRouter()

	mux.Use(middleware.RequestID)
	mux.Use(app.tracingMiddleware)
	mux.Use(middleware.RedirectSlashes)
	mux.Use(httprate.LimitByIP(180, time.Minute))
	mux.Use(cors.Handler(config))

	basePath := helpers.URLPath()

	mux.Handle(helpers.PathURL("/metrics"), promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))

	mux.Get(helpers.PathURL("/healthz"), func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if models.DB == nil || models.DB.PingContext(ctx) != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"down"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	fileServer := http.FileServer(http.Dir("presentation/public"))
	cacheStatic := func(next http.Handler, cacheControl string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cacheControl != "" {
				w.Header().Set("Cache-Control", cacheControl)
			}
			next.ServeHTTP(w, r)
		})
	}
	registerStaticRoutes := func(router chi.Router, routePrefix string, stripPrefix string) {
		router.Handle(routePrefix+"/css/*", cacheStatic(http.StripPrefix(stripPrefix, fileServer), "public, max-age=31536000, immutable"))
		router.Handle(routePrefix+"/js/*", cacheStatic(http.StripPrefix(stripPrefix, fileServer), "public, max-age=31536000, immutable"))
		router.Handle(routePrefix+"/icons/*", cacheStatic(http.StripPrefix(stripPrefix, fileServer), "public, max-age=31536000, immutable"))
		router.Handle(routePrefix+"/robots.txt", cacheStatic(http.StripPrefix(stripPrefix, fileServer), "public, max-age=3600"))
		router.Handle(routePrefix+"/sitemap.xml", cacheStatic(http.StripPrefix(stripPrefix, fileServer), "public, max-age=3600"))
		router.Handle(routePrefix+"/manifest.json", cacheStatic(http.StripPrefix(stripPrefix, fileServer), "public, max-age=3600"))
		router.Handle(routePrefix+"/sw.js", cacheStatic(http.StripPrefix(stripPrefix, fileServer), "no-cache, no-store, must-revalidate"))
	}

	registerStaticRoutes(mux, "", "/")
	if basePath != "" {
		registerStaticRoutes(mux, basePath, basePath+"/")
	}

	registerAppRoutes := func(router chi.Router) {
		indexController := controllers.IndexController{}
		indexController.RegisterRoutes(router)

		appController := controllers.NewAppController()
		appController.RegisterRoutes(router)

		profileHandler := profile.NewHandler(profile.NewService(profile.NewPostgresRepository()))
		profileHandler.RegisterRoutes(router)

		chatMVPHandler := mvpchat.NewHandler(mvpchat.NewService(mvpchat.NewPostgresRepository(), mvpchat.NewWebPushNotifierFromEnv()))
		chatMVPHandler.RegisterRoutes(router)
	}

	registerAppRoutes(mux)
	if basePath != "" {
		mux.Route(basePath, func(r chi.Router) {
			registerAppRoutes(r)
		})
	}

	if observability.LogsEnabled() {
		// Payment Module
		if app.DB == nil {
			fmt.Println("CRITICAL: app.DB is nil during router initialization!")
		}

		fmt.Println("Registered Routes:")
		chi.Walk(mux, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			fmt.Printf("[%s]: '%s'\n", method, route)
			return nil
		})
	}

	return mux
}

type loggingResponseWriter struct {
	middleware.WrapResponseWriter
	body bytes.Buffer
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	lrw.body.Write(b)
	return lrw.WrapResponseWriter.Write(b)
}

func (lrw *loggingResponseWriter) Flush() {
	if flusher, ok := lrw.WrapResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (app *App) tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &loggingResponseWriter{WrapResponseWriter: middleware.NewWrapResponseWriter(w, r.ProtoMajor)}

		var requestBody []byte
		contentType := r.Header.Get("Content-Type")
		isMultipart := strings.Contains(contentType, "multipart/form-data")

		if observability.DebugLoggingEnabled() && r.Body != nil && !isMultipart {
			requestBody, _ = io.ReadAll(io.LimitReader(r.Body, 4096))
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))

			entryBody := requestBody
			if len(entryBody) > 4096 {
				entryBody = append(entryBody[:4096], []byte("...")...)
			}

			observability.RequestEntryLogger().Printf(
				"%s %s start=%s body=%s",
				r.Method,
				r.URL.Path,
				start.Format(time.RFC3339Nano),
				bytes.TrimSpace(entryBody),
			)
		}

		next.ServeHTTP(wrapped, r)

		routePattern := r.URL.Path
		if routeContext := chi.RouteContext(r.Context()); routeContext != nil && routeContext.RoutePattern() != "" {
			routePattern = routeContext.RoutePattern()
		}

		statusCode := wrapped.Status()
		requestCounter.WithLabelValues(r.Method, routePattern, fmt.Sprintf("%d", statusCode)).Inc()
		requestID := middleware.GetReqID(r.Context())

		if statusCode >= http.StatusInternalServerError {
			observability.ErrorLogger().Printf(
				"request_id=%s method=%s path=%s route=%s status=%d duration=%s",
				requestID,
				r.Method,
				r.URL.Path,
				routePattern,
				statusCode,
				time.Since(start),
			)
		}

		if observability.DebugLoggingEnabled() {
			responseBody := wrapped.body.String()

			if len(requestBody) > 4096 {
				requestBody = append(requestBody[:4096], []byte("...")...)
			}

			if len(responseBody) > 4096 {
				responseBody = responseBody[:4096] + "..."
			}

			observability.RequestExitLogger().Printf(
				"request_id=%s %s %s status=%d duration=%s route=%s request_body=%s response_body=%s",
				requestID,
				r.Method,
				r.URL.Path,
				statusCode,
				time.Since(start).String(),
				routePattern,
				bytes.TrimSpace(requestBody),
				responseBody,
			)
		}
	})
}
