package market

import (
	"context"
	"errors"
	"fmt"
	"seer/internal/numeric"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	query := `INSERT INTO ledger_accounts(account_type, currency, allow_negative_balance, allow_positive_balance)
	VALUES('house', 'USDT', false, true)
	RETURNING id`

	err = tx.QueryRow(ctx, query).Scan(&m.HouseLedgerAccountID)
	if err != nil {
		return fmt.Errorf("failed to create house ledger account: %w", err)
	}

	// Create the market
	query = `INSERT INTO markets(status, 
	name, description, currency, img_key,
	house_ledger_account_id, q0_seeding, alpha, fee, cap_price,
	outcome_sort,
	close_time)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	RETURNING id`

	err = tx.QueryRow(ctx, query, StatusDraft,
		m.Name, m.Description, m.Currency, m.ImgKey,
		m.HouseLedgerAccountID, m.Q0Seeding, m.Alpha, m.Fee, m.CapPrice,
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

func (ma *AdminManager) GetMarketStatus(ctx context.Context, marketID uuid.UUID) (MarketStatus, error) {

	var status MarketStatus

	query := `SELECT status FROM markets WHERE id = $1`

	err := ma.db.QueryRow(ctx, query, marketID).Scan(&status)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return "", ErrMarketNotFound
		default:
			return "", fmt.Errorf("failed to get market status: %w", err)
		}
	}
	return status, nil
}

func (ma *AdminManager) UpdateMarketFees(ctx context.Context, marketID uuid.UUID, newFee *numeric.BigDecimal) error {

	cmd, err := ma.db.Exec(ctx, `UPDATE markets SET fee = $1 WHERE id = $2`, newFee, marketID)

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

func (ma *AdminManager) UpdateOutcomeSort(ctx context.Context, marketID uuid.UUID, outcomeSort MarketOutcomeSort) error {
	cmd, err := ma.db.Exec(ctx, `UPDATE markets SET outcome_sort = $1 WHERE id = $2`, outcomeSort, marketID)

	if err != nil {
		return fmt.Errorf("failed to update outcome sort: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrMarketNotFound
	}

	return nil
}

func (ma *AdminManager) UpdateOutcomePositions(ctx context.Context, marketID uuid.UUID, outcomes []*Outcome) error {

	query := `UPDATE outcomes SET position = $1 WHERE id = $2 AND market_id = $3`

	tx, err := ma.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	for _, o := range outcomes {
		cmd, err := tx.Exec(ctx, query, o.Position, o.ID, marketID)
		if err != nil {
			return fmt.Errorf("failed to update outcome %d position: %w", o.ID, err)
		}
		if cmd.RowsAffected() == 0 {
			return fmt.Errorf("outcome %d not found for market %s", o.ID, marketID)
		}
	}

	return tx.Commit(ctx)
}
