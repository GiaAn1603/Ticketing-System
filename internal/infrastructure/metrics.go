package infrastructure

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TicketSales = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ticket_sales_total",
			Help: "Total ticket purchase attempts",
		},
		[]string{"event_id", "status"},
	)

	RateLimitRejections = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limit_rejections_total",
			Help: "Total requests rejected by rate limiter",
		},
	)
)
