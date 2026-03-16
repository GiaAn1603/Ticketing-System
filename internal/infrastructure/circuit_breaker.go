package infrastructure

import (
	"log/slog"
	"time"

	"github.com/sony/gobreaker"
)

func NewCircuitBreaker(
	logger *slog.Logger,
	name string,
	maxReq, minReq uint32,
	failRatio float64,
	interval, timeout time.Duration,
) *gobreaker.CircuitBreaker {
	st := gobreaker.Settings{
		Name:        name,
		MaxRequests: maxReq,
		Interval:    interval,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= minReq && failureRatio >= failRatio
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			switch to {
			case gobreaker.StateOpen:
				logger.Error(
					"Circuit breaker tripped to open",
					"cb_name", name,
				)
			case gobreaker.StateClosed:
				logger.Info(
					"Circuit breaker closed successfully",
					"cb_name", name,
				)
			}
		},
	}

	return gobreaker.NewCircuitBreaker(st)
}
