package market

import (
	"context"
	"errors"
	"fmt"
	"seer/internal/finance"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const minBetAmount = 1_00      // 1 USDT
const maxBetAmount = 10_000_00 // 10k USDT

type TransactionManager struct {
	db *pgxpool.Pool
}

type BetRequest struct {
	UserID          uuid.UUID
	MarketID        uuid.UUID
	OutcomeID       int64
	BetAmountCents  int64
	QuotedGainCents int64
	Currency        string
	IdempotencyKey  string
}

const maxAttempts = 3

func NewTransactionManager(db *pgxpool.Pool) *TransactionManager {
	return &TransactionManager{
		db: db,
	}
}

func (tm *TransactionManager) AddBet(ctx context.Context, r BetRequest) error {

	if r.BetAmountCents < minBetAmount || r.BetAmountCents > maxBetAmount {
		return ErrInvalidBetAmount
	}

	if r.QuotedGainCents <= r.BetAmountCents {
		return errors.New("quoted gain must be > bet amount")
	}

	for attempt := range maxAttempts {

		err := tm.addBetOnce(ctx, r)
		if err == nil {
			return nil
		}

		var pgErr *pgconn.PgError
		isRetryable := errors.As(err, &pgErr) && (pgErr.Code == pgerrcode.SerializationFailure || pgErr.Code == pgerrcode.DeadlockDetected)

		if !isRetryable {
			return err
		}

		if attempt < maxAttempts-1 {
			select {
			// Larger backoff at each retry
			case <-time.After(time.Duration(50*(attempt+1)) * time.Millisecond):
			// Context is done, exit early
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		return err
	}

	return errors.New("too many serialization retries")
}

func (tm *TransactionManager) addBetOnce(ctx context.Context, r BetRequest) error {

	// Begin a serializable transaction
	opts := pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	}

	tx, err := tm.db.BeginTx(ctx, opts)

	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	defer tx.Rollback(ctx)

	// Retrieve the user's account
	userLedgerAccountID, err := finance.GetUserAccountForCurrency(ctx, tx, r.UserID, r.Currency)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Retrieve the market's current state -> prices, fee etc
	// -> We want to recompute the price

	var m Market
	var outcomesIDs, qVec []int64

	query := `
	SELECT m.id, m.house_ledger_account_id, m.alpha_ppm, m.fee_ppm, m.version,
	array_agg(o.quantity ORDER BY o.id) AS q_vec,
	array_agg(o.id ORDER BY o.id) AS outcome_ids
	FROM markets m
	JOIN outcomes o ON o.market_id = m.id
	WHERE m.id = $1 AND m.status = 'opened' AND (m.close_time IS NULL OR m.close_time > now())
	GROUP BY m.id`

	if err = tx.QueryRow(ctx, query, r.MarketID).Scan(&m.ID, &m.HouseLedgerAccountID, &m.AlphaPPM, &m.FeePPM, &m.Version, &qVec, &outcomesIDs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMarketNotFound
		}
		return fmt.Errorf("failed to query market's current state: %w", err)
	}

	if len(qVec) != len(outcomesIDs) || len(qVec) == 0 {
		return errors.New("inconsistent outcomes for market")
	}

	// Find the index of the outcome in the q vector
	idx := slices.Index(outcomesIDs, r.OutcomeID)
	if idx == -1 {
		return ErrOutcomeNotFound
	}

	// Recompute gain in cents for the user
	// Validate it's equal to the provided BetRequest
	actualGainCents, feeCents, _, err := OddAndGainFromBudget(qVec, m.AlphaPPM, m.FeePPM, r.BetAmountCents, idx)
	if err != nil {
		return fmt.Errorf("failed to recompute actual gain: %w", err)
	}

	if actualGainCents != r.QuotedGainCents {
		return ErrInvalidQuotedGain
	}

	// Everything is valid, we can start comiting

	// Make transaction
	err = finance.TransferMoney(ctx, tx, userLedgerAccountID, m.HouseLedgerAccountID, r.BetAmountCents, r.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("failed to transfer: %w", err)
	}

	// Commit a bet, verify time with close_time > now()
	bet := Bet{
		LedgerAccountID:     userLedgerAccountID,
		OutcomeID:           r.OutcomeID,
		PayoutCents:         actualGainCents,
		TotalPricePaidCents: r.BetAmountCents,
		FeePaidCents:        feeCents,
		FeePPM:              m.FeePPM,
		IdempotencyKey:      r.IdempotencyKey,
	}

	query = `
	INSERT INTO bets(ledger_account_id, outcome_id, payout_cents, total_price_paid_cents, fee_paid_cents, fee_ppm, idempotency_key)
	VALUES($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (idempotency_key) DO NOTHING
	`

	cmd, err := tx.Exec(ctx, query, bet.LedgerAccountID, bet.OutcomeID, bet.PayoutCents, bet.TotalPricePaidCents, bet.FeePaidCents, bet.FeePPM, bet.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("update outcome position: %w", err)
	}

	// Already have a bet with this idempotency key
	if cmd.RowsAffected() == 0 {
		return ErrBetAlreadyPlaced
	}

	// Add the shares to the outcome
	if _, err := tx.Exec(ctx, `UPDATE outcomes SET quantity = quantity + $1, volume_cents = volume_cents + $2 WHERE id = $3`, actualGainCents, r.BetAmountCents, r.OutcomeID); err != nil {
		return fmt.Errorf("update outcome position: %w", err)
	}

	// Update the market's version
	cmd, err = tx.Exec(ctx, `UPDATE markets SET version = version + 1, volume_cents = volume_cents + $1 WHERE id=$2 AND (close_time IS NULL OR close_time > now())`, r.BetAmountCents, r.MarketID)

	if err != nil {
		return fmt.Errorf("failed to update market version: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrMarketNotFound
	}

	return tx.Commit(ctx)

}

// TODO -> settle market (takes an outcome, pay relevant shares)
func (tm *TransactionManager) SettleMarket(ctx context.Context, marketID uuid.UUID, winningOutcomeID int64) error {

	tx, err := tm.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	defer tx.Rollback(ctx)

	// Make sure the outcome is tied to this market
	var outcomeMarketID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT market_id FROM outcomes WHERE id = $1`, winningOutcomeID).Scan(&outcomeMarketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOutcomeNotFound
		}
		return err
	}

	if outcomeMarketID != marketID {
		return errors.New("winning outcome doesn't belong to market")
	}

	// Change the market status to 'settling' (should have been done earlier but re-done here for consistance)

	var houseAccountID uuid.UUID
	query := `UPDATE markets 
	SET status = 'settling' 
	WHERE id = $1 AND status IN ('opened','paused')
	RETURNING house_ledger_account_id
	`
	err = tx.QueryRow(ctx, query, marketID).Scan(&houseAccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMarketNotFound
		}
		return fmt.Errorf("failed to change market status to settling: %w", err)
	}

	// We have to move funds from the house account
	// Pay winners [-> make sure enough funds on tied house account to the market]
	// Remaining funds -> don't touch them

	// Insert the resolution row
	cmd, err := tx.Exec(ctx, `INSERT INTO market_resolutions (market_id, winning_outcome_id) 
	VALUES ($1,$2) ON CONFLICT DO NOTHING`, marketID, winningOutcomeID)

	if err != nil {
		return fmt.Errorf("failed to insert resolution: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return errors.New("market already resolved")
	}

	// Compute the amount to pay to each account
	query = `
	SELECT b.ledger_account_id, b.payout_cents	
	FROM bets b
	JOIN outcomes o ON b.outcome_id = o.id -- NOT ESSENTIEL, just to mitigate the ability to choose an outcome not tied to the given marketID
	WHERE o.market_id = $1 AND b.outcome_id = $2
	GROUP BY b.ledger_account_id;
	`

	rows, err := tx.Query(ctx, query, marketID, winningOutcomeID)

	if err != nil {
		return fmt.Errorf("failed to aggregate payouts: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var ledgerAccountID uuid.UUID
		var payoutCents int64

		err = rows.Scan(&ledgerAccountID, &payoutCents)
		if err != nil {
			return fmt.Errorf("failed to scan: %w", err)
		}

		idemKey := fmt.Sprintf("settle:%s:%s", marketID, ledgerAccountID)
		err = finance.TransferMoney(ctx, tx, houseAccountID, ledgerAccountID, payoutCents, idemKey)
		if err != nil {
			return fmt.Errorf("failed to credit account %s: %w", ledgerAccountID, err)
		}
	}

	if rows.Err() != nil {
		return rows.Err()
	}

	query = `UPDATE markets 
	SET status = 'resolved', version=version+1 
	WHERE id=$1`

	_, err = tx.Exec(ctx, query, marketID)
	if err != nil {
		return fmt.Errorf("failed to finalize settlement: %w", err)
	}

	return tx.Commit(ctx)
}

func (tm *TransactionManager) CancelMarket(ctx context.Context, marketID uuid.UUID) error {
	tx, err := tm.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	defer tx.Rollback(ctx)

	// Change the market status to 'settling' (should have been done earlier but re-done here for consistency)
	var houseAccountID uuid.UUID
	query := `UPDATE markets 
	SET status = 'settling' 
	WHERE id = $1 AND status IN ('opened','paused')
	RETURNING house_ledger_account_id
	`

	err = tx.QueryRow(ctx, query, marketID).Scan(&houseAccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMarketNotFound
		}
		return fmt.Errorf("failed to change market status to settling: %w", err)
	}

	// Insert the cancellation row
	cmd, err := tx.Exec(ctx, `INSERT INTO market_cancellations(market_id) 
	VALUES ($1) ON CONFLICT DO NOTHING`, marketID)

	if err != nil {
		return fmt.Errorf("failed to insert resolution: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return errors.New("market already cancelled")
	}

	// Compute the amount to refund to each account
	query = `
	SELECT b.ledger_account_id, SUM(b.total_price_paid_cents) as refund_cents	
	FROM bets b
	JOIN outcomes o ON b.outcome_id = o.id -- NOT ESSENTIAL, just to mitigate the ability to choose an outcome not tied to the given marketID
	WHERE o.market_id = $1
	GROUP BY b.ledger_account_id;
	`

	rows, err := tx.Query(ctx, query, marketID)

	if err != nil {
		return fmt.Errorf("failed to aggregate refunds: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var ledgerAccountID uuid.UUID
		var refundCents int64

		err = rows.Scan(&ledgerAccountID, &refundCents)
		if err != nil {
			return fmt.Errorf("failed to scan")
		}

		idemKey := fmt.Sprintf("settle:%s:%s", marketID, ledgerAccountID)
		err = finance.TransferMoney(ctx, tx, houseAccountID, ledgerAccountID, refundCents, idemKey)
		if err != nil {
			return fmt.Errorf("failed to refund account %s: %w", ledgerAccountID, err)
		}
	}

	if rows.Err() != nil {
		return rows.Err()
	}

	query = `UPDATE markets 
	SET status = 'cancelled', version=version+1 
	WHERE id=$1`

	_, err = tx.Exec(ctx, query, marketID)
	if err != nil {
		return fmt.Errorf("failed to finalize settlement: %w", err)
	}

	return tx.Commit(ctx)
}
