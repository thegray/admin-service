package prometheus

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	requests  *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	startTime time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{
		requests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "admin-service_http_requests_total",
			Help: "Count of HTTP requests handled by the admin-service service.",
		}, []string{"method", "path", "status"}),
		duration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "admin-service_http_request_duration_seconds",
			Help:    "Duration of HTTP requests handled by admin-service.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		startTime: time.Now(),
	}
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		status := strconv.Itoa(c.Writer.Status())

		m.requests.WithLabelValues(c.Request.Method, path, status).Inc()
		m.duration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}

func (m *Metrics) Handler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}
