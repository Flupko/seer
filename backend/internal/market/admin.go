package market

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminManager struct {
	db *pgxpool.Pool
}

func NewAdminManager(db *pgxpool.Pool) *AdminManager {
	return &AdminManager{
		db: db,
	}
}

func (am *AdminManager) CreateMarket(ctx context.Context, m *Market, categoryIDs []int64, outcomes []*Outcome) error {

	tx, err := am.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert a new house ledger specially for this market
	query := `INSERT INTO ledger_accounts(account_type, currency)
	VALUES('house', 'USDT')
	RETURNING id`

	err = tx.QueryRow(ctx, query).Scan(&m.HouseLedgerAccountID)
	if err != nil {
		return fmt.Errorf("failed to create house ledger account: %w", err)
	}

	// Create the market
	query = `INSERT INTO markets(status, 
	name, description, 
	house_ledger_account_id, q0_seeding, alpha_ppm, fee_ppm, 
	outcome_sort,
	close_time)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
	RETURNING id`

	err = tx.QueryRow(ctx, query, StatusDraft,
		m.Name, m.Description,
		m.HouseLedgerAccountID, m.Q0Seeding, m.AlphaPPM, m.FeePPM,
		m.OutcomeSort,
		m.CloseTime).Scan(&m.ID)
	if err != nil {
		return err
	}

	// Add the categories to it

	if len(categoryIDs) > 0 {
		query = `INSERT INTO categories_market(market_id, category_id)
		VALUES($1, $2)`

		for _, catID := range categoryIDs {
			if _, err = tx.Exec(ctx, query, m.ID, catID); err != nil {
				return fmt.Errorf("failed to insert market category %d: %w", catID, err)
			}
		}

	}

	// Add the outcomes to the market
	query = `INSERT INTO outcomes(market_id, name, position, quantity)
	VALUES($1, $2, $3, $4) 
	`

	for _, o := range outcomes {
		_, err := tx.Exec(ctx, query, m.ID, o.Name, o.Position, m.Q0Seeding)
		if err != nil {
			return fmt.Errorf("failed to insert outcome: %w", err)
		}
	}

	return tx.Commit(ctx)

}

func (ma *AdminManager) UpdateMarketFees(ctx context.Context, marketID uuid.UUID, newFeePPM int64) error {

	cmd, err := ma.db.Exec(ctx, `UPDATE markets SET fee_ppm = $1 WHERE id = $2`, newFeePPM, marketID)

	if err != nil {
		return fmt.Errorf("failed to update fee: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrMarketNotFound
	}

	return nil
}

func (ma *AdminManager) PauseMarket(ctx context.Context, marketID uuid.UUID) error {
	cmd, err := ma.db.Exec(ctx, `UPDATE markets SET status = $1 WHERE id = $2`, StatusPaused, marketID)

	if err != nil {
		return fmt.Errorf("failed to update fee: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrMarketNotFound
	}

	return nil
}

func (ma *AdminManager) ResumeMarket(ctx context.Context, marketID uuid.UUID) error {
	cmd, err := ma.db.Exec(ctx, `UPDATE markets SET status = $1 WHERE id = $2`, StatusOpened, marketID)

	if err != nil {
		return fmt.Errorf("failed to update fee: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrMarketNotFound
	}

	return nil
}

func (ma *AdminManager) UpdateMarketCloseTime(ctx context.Context, marketID uuid.UUID, closeTime time.Time) error {
	cmd, err := ma.db.Exec(ctx, `UPDATE markets SET close_time = $1 WHERE id = $2`, closeTime, marketID)

	if err != nil {
		return fmt.Errorf("failed to update fee: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrMarketNotFound
	}

	return nil
}
