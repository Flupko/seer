package market

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VolumeCalculator struct {
	db     *pgxpool.Pool
	logger *slog.Logger

	updateInterval time.Duration
	maxRetries     int64
}

func NewVolumeCalculator(db *pgxpool.Pool, logger *slog.Logger, interval time.Duration) *VolumeCalculator {
	return &VolumeCalculator{
		db:             db,
		logger:         logger,
		updateInterval: interval,
		maxRetries:     3,
	}
}

func (vc *VolumeCalculator) Start(ctx context.Context) {

	ticker := time.NewTicker(vc.updateInterval)
	defer ticker.Stop()

	vc.logger.Info("starting volume calculator", "interval", vc.updateInterval)

	// Initial update
	if err := vc.updateVolume24h(ctx); err != nil {
		vc.logger.Error("initial volume update failed", "error", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := vc.updateVolume24h(ctx); err != nil {
				vc.logger.Error("volume update failed", "error", err)
			}
		case <-ctx.Done():
			vc.logger.Info("exiting volume calculator")
			return
		}
	}
}

func (vc *VolumeCalculator) updateVolume24h(ctx context.Context) error {

	query := `UPDATE markets m
	SET volume_24h = (
		SELECT COALESCE(SUM(b.total_price_paid_cents), 0)
		FROM bets tb
		JOIN outcomes o ON b.outcome_id = o.id 
		WHERE o.market_id = m.id 
		AND b.purchase_time >= NOW() - INTERVAL '24 hours'
	);
	`

	vc.logger.Info("starting market volume update")

	for attempt := range vc.maxRetries {

		// 2 minutes to compute
		updateCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		_, err := vc.db.Exec(updateCtx, query)
		cancel()

		if err == nil {
			vc.logger.Info("volume update completed", "attempt", attempt+1)
			return nil
		}

		vc.logger.Error("volume update attempt failed",
			"attempt", attempt+1,
			"error", err)

		if attempt < vc.maxRetries-1 {

			select {
			// Larger backoff at each retry
			case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
			// Context is done, exit early
			case <-ctx.Done():
				return ctx.Err()
			}
		}

	}

	return fmt.Errorf("volume update failed after %d attempts", vc.maxRetries)

}
