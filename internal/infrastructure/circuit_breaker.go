package infrastructure

import (
	"Ticketing-System/internal/config"
	"log/slog"

	"github.com/sony/gobreaker"
)

func NewCircuitBreaker(logger *slog.Logger, name string, cfg config.CircuitBreakerConfig) *gobreaker.CircuitBreaker {
	st := gobreaker.Settings{
		Name:        name,
		MaxRequests: cfg.MaxReq,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= cfg.MinReq && failureRatio >= cfg.FailRatio
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
