package worker

import (
	"context"
	"time"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/repository"
	"github.com/ecotracker/backend/internal/service"
	"github.com/sirupsen/logrus"
)

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

func (w *AssignmentWorker) Start() {
	go func() {
		logrus.WithField("interval", w.interval).Info("AssignmentWorker dimulai")
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

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

func (w *AssignmentWorker) Stop() {
	select {
	case <-w.done:
	default:
		close(w.done)
	}
}

func (w *AssignmentWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// 1. Handle expired assignments (sudah ada sebelumnya)
	w.handleExpiredAssignments(ctx)

	// 2. Handle pending pickups yang belum dapat collector ← INI YANG BARU
	w.handlePendingPickups(ctx)
}

func (w *AssignmentWorker) handleExpiredAssignments(ctx context.Context) {
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
			"reassignment_count": pickup.ReassignmentCount,
		})

		if pickup.ReassignmentCount >= 5 {
			log.Warn("Pickup melewati batas reassignment, kembali ke pending")
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

// handlePendingPickups mencoba assign pickup yang masih pending ke collector yang baru online
func (w *AssignmentWorker) handlePendingPickups(ctx context.Context) {
	pendingPickups, err := w.pickupRepo.FindPendingPickups(ctx)
	if err != nil {
		logrus.WithError(err).Error("AssignmentWorker: gagal mencari pending pickups")
		return
	}

	if len(pendingPickups) == 0 {
		return
	}

	logrus.WithField("jumlah", len(pendingPickups)).Info("AssignmentWorker: mencoba assign pending pickups")

	for i := range pendingPickups {
		pickup := pendingPickups[i]
		log := logrus.WithField("pickup_id", pickup.ID)

		// Gunakan AssignPickup langsung — sama seperti saat pickup pertama dibuat
		if err := w.assignmentService.AssignClosestCollector(ctx, pickup.ID, pickup.Lat, pickup.Lon, []string{}); err != nil {
			log.WithField("reason", err.Error()).Debug("Pending pickup belum bisa di-assign")
		} else {
			log.Info("Pending pickup berhasil di-assign ke collector baru")
		}
	}
}