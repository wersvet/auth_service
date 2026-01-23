package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	requestsTotalName    = "http_requests_total"
	requestDurationName  = "http_request_duration_seconds"
	inFlightRequestsName = "http_in_flight_requests"
	authLoginsTotalName  = "auth_logins_total"
	authRegistersName    = "auth_registers_total"
	statusLabel          = "status"
	methodLabel          = "method"
	pathLabel            = "path"
	serviceLabel         = "service"
)

const (
	AuthStatusSuccess = "success"
	AuthStatusFailed  = "failed"
)

type Metrics struct {
	service          string
	once             sync.Once
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	inFlightRequests prometheus.Gauge
	authLoginsTotal  *prometheus.CounterVec
	authRegisters    *prometheus.CounterVec
	registry         prometheus.Registerer
	gatherer         prometheus.Gatherer
}

func New(service string) *Metrics {
	return NewWithRegistry(service, prometheus.DefaultRegisterer, prometheus.DefaultGatherer)
}

func NewWithRegistry(service string, registry prometheus.Registerer, gatherer prometheus.Gatherer) *Metrics {
	return &Metrics{
		service:  service,
		registry: registry,
		gatherer: gatherer,
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: requestsTotalName,
			Help: "Total number of HTTP requests processed.",
		}, []string{serviceLabel, methodLabel, pathLabel, statusLabel}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    requestDurationName,
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{serviceLabel, methodLabel, pathLabel, statusLabel}),
		inFlightRequests: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: inFlightRequestsName,
			Help: "Current number of in-flight HTTP requests.",
		}),
		authLoginsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: authLoginsTotalName,
			Help: "Total number of authentication login attempts.",
		}, []string{statusLabel}),
		authRegisters: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: authRegistersName,
			Help: "Total number of authentication registrations.",
		}, []string{statusLabel}),
	}
}

func (m *Metrics) Register() {
	m.once.Do(func() {
		m.registry.MustRegister(
			m.requestsTotal,
			m.requestDuration,
			m.inFlightRequests,
			m.authLoginsTotal,
			m.authRegisters,
		)
	})
}

func (m *Metrics) Handler() http.Handler {
	if m.gatherer == prometheus.DefaultGatherer {
		return promhttp.Handler()
	}
	return promhttp.HandlerFor(m.gatherer, promhttp.HandlerOpts{})
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	m.Register()

	return func(c *gin.Context) {
		start := time.Now()
		m.inFlightRequests.Inc()
		defer m.inFlightRequests.Dec()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		labels := prometheus.Labels{
			serviceLabel: m.service,
			methodLabel:  c.Request.Method,
			pathLabel:    path,
			statusLabel:  status,
		}

		m.requestsTotal.With(labels).Inc()
		m.requestDuration.With(labels).Observe(time.Since(start).Seconds())
	}
}

func (m *Metrics) IncAuthLogin(status string) {
	if m == nil || !isValidAuthStatus(status) {
		return
	}
	m.Register()
	m.authLoginsTotal.WithLabelValues(status).Inc()
}

func (m *Metrics) IncAuthRegister(status string) {
	if m == nil || !isValidAuthStatus(status) {
		return
	}
	m.Register()
	m.authRegisters.WithLabelValues(status).Inc()
}

func isValidAuthStatus(status string) bool {
	return status == AuthStatusSuccess || status == AuthStatusFailed
}
