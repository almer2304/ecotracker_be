package worker

import (
	"context"
	"time"

	"github.com/ecotracker/backend/internal/service"
	"github.com/sirupsen/logrus"
)

// TimeoutChecker - digantikan oleh AssignmentWorker
// Dipertahankan agar tidak ada unused import error
type TimeoutChecker struct {
	assignmentService *service.AssignmentService
	interval          time.Duration
	done              chan struct{}
}

func NewTimeoutChecker(
	assignmentService *service.AssignmentService,
	interval time.Duration,
) *TimeoutChecker {
	return &TimeoutChecker{
		assignmentService: assignmentService,
		interval:          interval,
		done:              make(chan struct{}),
	}
}

func (w *TimeoutChecker) Start() {
	go func() {
		logrus.Info("TimeoutChecker dimulai (delegasi ke AssignmentWorker)")
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Ditangani oleh AssignmentWorker
			case <-w.done:
				return
			}
		}
	}()
}

func (w *TimeoutChecker) Stop() {
	select {
	case <-w.done:
	default:
		close(w.done)
	}
}

func getBackgroundCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}