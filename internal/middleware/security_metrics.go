package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	authFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "microvibe_security_auth_failures_total",
			Help: "认证失败次数，按端点和原因分类",
		},
		[]string{"endpoint", "reason"},
	)

	rateLimitHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "microvibe_security_ratelimit_hits_total",
			Help: "速率限制命中次数，按端点分类",
		},
		[]string{"endpoint"},
	)

	corsBlockedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "microvibe_security_cors_blocked_total",
			Help: "CORS 拦截请求数，按来源分类",
		},
		[]string{"origin"},
	)

	tokenValidationFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "microvibe_security_token_validation_failures_total",
			Help: "Token 验证失败次数，按原因分类",
		},
		[]string{"reason"},
	)

	activeBlockedIPs = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "microvibe_security_active_blocked_ips",
			Help: "当前被封禁的 IP 数量，按端点分类",
		},
		[]string{"endpoint"},
	)
)
