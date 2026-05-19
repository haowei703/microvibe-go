package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// httpRequestCount 请求总数计数器
	httpRequestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "microvibe_http_requests_total",
			Help: "HTTP 请求总数",
		},
		[]string{"method", "path", "status"},
	)

	// httpRequestDuration 请求耗时分布（毫秒）
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "microvibe_http_request_duration_ms",
			Help:    "HTTP 请求耗时分布（毫秒）",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 150, 200, 300, 500, 1000, 2000, 5000},
		},
		[]string{"method", "path"},
	)

	// httpRequestDurationSummary P50/P90/P99 百分位
	httpRequestDurationSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "microvibe_http_request_duration_summary_ms",
			Help:       "HTTP 请求耗时百分位统计（ms）",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"method", "path"},
	)

	// httpInFlight 当前正在处理的请求数
	httpInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "microvibe_http_in_flight_requests",
			Help: "当前正在处理的 HTTP 请求数",
		},
	)

	// httpResponseSize 响应大小分布（字节）
	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "microvibe_http_response_size_bytes",
			Help:    "HTTP 响应大小分布（字节）",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6),
		},
		[]string{"method", "path"},
	)
)

// PrometheusMiddleware 采集 HTTP 指标
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过 /metrics 自身的指标采集
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		httpInFlight.Inc()

		c.Next()

		httpInFlight.Dec()

		elapsed := float64(time.Since(start).Milliseconds())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())

		httpRequestCount.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(elapsed)
		httpRequestDurationSummary.WithLabelValues(method, path).Observe(elapsed)
		httpResponseSize.WithLabelValues(method, path).Observe(float64(c.Writer.Size()))
	}
}
