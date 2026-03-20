package worker

import (
	"context"
	"time"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/repository"
	"github.com/ecotracker/backend/internal/service"
	"github.com/sirupsen/logrus"
)

// AssignmentWorker adalah background worker lengkap yang mengelola timeout & reassignment
type AssignmentWorker struct {
	assignmentService *service.AssignmentService
	pickupRepo        *repository.PickupRepository
	interval          time.Duration
	done              chan struct{}
}

func NewAssignmentWorker(
	assignmentService *service.AssignmentService,
	pickupRepo *repository.PickupRepository,
	interval time.Duration,
) *AssignmentWorker {
	return &AssignmentWorker{
		assignmentService: assignmentService,
		pickupRepo:        pickupRepo,
		interval:          interval,
		done:              make(chan struct{}),
	}
}

// Start menjalankan worker di background goroutine
func (w *AssignmentWorker) Start() {
	go func() {
		logrus.WithField("interval", w.interval).Info("AssignmentWorker dimulai")
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		// Jalankan sekali langsung saat startup
		w.run()

		for {
			select {
			case <-ticker.C:
				w.run()
			case <-w.done:
				logrus.Info("AssignmentWorker dihentikan")
				return
			}
		}
	}()
}

// Stop menghentikan worker secara graceful
func (w *AssignmentWorker) Stop() {
	select {
	case <-w.done:
		// Sudah ditutup
	default:
		close(w.done)
	}
}

// run mengecek expired assignments dan melakukan reassignment
func (w *AssignmentWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Temukan pickup yang sudah timeout
	expiredPickups, err := w.pickupRepo.FindExpiredAssignments(ctx)
	if err != nil {
		logrus.WithError(err).Error("AssignmentWorker: gagal mencari expired assignments")
		return
	}

	if len(expiredPickups) == 0 {
		return
	}

	logrus.WithField("jumlah", len(expiredPickups)).Info("AssignmentWorker: memproses expired pickups")

	for i := range expiredPickups {
		pickup := expiredPickups[i]

		log := logrus.WithFields(logrus.Fields{
			"pickup_id":          pickup.ID,
			"collector_id":       pickup.CollectorID,
			"reassignment_count": pickup.ReassignmentCount,
		})

		// Batasi maksimum 5x reassignment untuk mencegah infinite loop
		if pickup.ReassignmentCount >= 5 {
			log.Warn("Pickup telah melewati batas reassignment, ubah ke pending")
			w.pickupRepo.UpdateStatus(ctx, pickup.ID, domain.StatusPending, nil)
			continue
		}

		if err := w.assignmentService.ReassignPickup(ctx, pickup); err != nil {
			log.WithError(err).Error("Gagal reassign pickup")
		} else {
			log.Info("Pickup berhasil di-reassign")
		}
	}
}