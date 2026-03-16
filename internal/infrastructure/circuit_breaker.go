package infrastructure

import (
	"log/slog"
	"time"

	"github.com/sony/gobreaker"
)

func NewCircuitBreaker(name string, maxReq uint32, interval, timeout time.Duration, logger *slog.Logger) *gobreaker.CircuitBreaker {
	st := gobreaker.Settings{
		Name:        name,
		MaxRequests: maxReq,
		Interval:    interval,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			switch to {
			case gobreaker.StateOpen:
				logger.Error(
					"Circuit breaker opened",
					KeyAction, "circuit_breaker_trip",
					KeyStatus, StatusFailed,
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
