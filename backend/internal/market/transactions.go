package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"seer/internal/finance"
	"seer/internal/numeric"
	"seer/internal/ps"
	"slices"
	"time"

	"github.com/ericlagergren/decimal"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var minBetAmount = decimal.New(1, 1)     // 0.1 USDT
var maxBetAmount = decimal.New(10000, 0) // 10k USDT

type TransactionManager struct {
	rdb    *redis.Client
	db     *pgxpool.Pool
	logger *slog.Logger
}

type BetRequest struct {
	LedgerAccountID uuid.UUID
	UserID          uuid.UUID
	MarketID        uuid.UUID
	OutcomeID       int64
	BetAmount       *numeric.BigDecimal
	MinWantedGain   *numeric.BigDecimal
	Currency        finance.Currency
	IdempotencyKey  string
}

const maxAttempts = 3

func NewTransactionManager(rdb *redis.Client, db *pgxpool.Pool, logger *slog.Logger) *TransactionManager {
	return &TransactionManager{
		rdb:    rdb,
		db:     db,
		logger: logger,
	}
}

func (tm *TransactionManager) AddBet(ctx context.Context, r BetRequest) (*Bet, error) {

	if r.BetAmount.Cmp(minBetAmount) < 0 || r.BetAmount.Cmp(maxBetAmount) > 0 {
		return nil, ErrInvalidBetAmount
	}

	if r.MinWantedGain.Cmp(&r.BetAmount.Big) <= 0 {
		fmt.Println("invalid quoted gain:", r.MinWantedGain.String(), "bet amount:", r.BetAmount.String())
		return nil, ErrInvalidQuotedGain
	}

	for attempt := range maxAttempts {

		bet, err := tm.addBetOnce(ctx, r)
		if err == nil {
			// Push market update to redis
			go func() {
				if err := tm.PulishUpdateMarket(r.MarketID); err != nil {
					tm.logger.Error("failed to publish market update", "error", err)
				}

				if err := tm.PublishBalanceUpdate(r.LedgerAccountID); err != nil {
					tm.logger.Error("failed to publish balance update", "error", err)
				}

				if err := tm.PublishBetUpdate(bet.ID); err != nil {
					tm.logger.Error("failed to publish bet update", "error", err)
				}

			}()
			return bet, nil
		}

		var pgErr *pgconn.PgError
		isRetryable := errors.As(err, &pgErr) && (pgErr.Code == pgerrcode.SerializationFailure || pgErr.Code == pgerrcode.DeadlockDetected)

		if !isRetryable {
			return nil, err
		}

		if attempt < maxAttempts-1 {
			select {
			// Larger backoff at each retry
			case <-time.After(time.Duration(50*(attempt+1)) * time.Millisecond):
			// Context is done, exit early
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		return nil, err
	}

	return nil, errors.New("too many serialization retries")
}

func (tm *TransactionManager) addBetOnce(ctx context.Context, r BetRequest) (*Bet, error) {

	// Begin a serializable transaction
	opts := pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	}

	tx, err := tm.db.BeginTx(ctx, opts)

	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}

	defer tx.Rollback(ctx)

	// Retrieve the market's current state -> prices, fee etc
	// -> We want to recompute the price

	var m Market
	var outcomesIDs []int64
	var qVec []*numeric.BigDecimal

	query := `
	SELECT m.id, m.house_ledger_account_id, m.alpha, m.fee, m.cap_price,
	m.version,
	array_agg(o.quantity ORDER BY o.id) AS q_vec,
	array_agg(o.id ORDER BY o.id) AS outcome_ids
	FROM markets m
	JOIN outcomes o ON o.market_id = m.id
	WHERE m.id = $1 AND m.status = 'opened' AND (m.close_time IS NULL OR m.close_time > now())
	GROUP BY m.id`

	if err = tx.QueryRow(ctx, query, r.MarketID).Scan(&m.ID, &m.HouseLedgerAccountID, &m.Alpha, &m.Fee, &m.CapPrice, &m.Version, &qVec, &outcomesIDs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMarketNotFound
		}
		return nil, fmt.Errorf("failed to query market's current state: %w", err)
	}

	if len(qVec) != len(outcomesIDs) || len(qVec) == 0 {
		return nil, errors.New("inconsistent outcomes for market")
	}

	// Find the index of the outcome in the q vector
	idx := slices.Index(outcomesIDs, r.OutcomeID)
	if idx == -1 {
		return nil, ErrOutcomeNotFound
	}

	query = `
        SELECT balance
        FROM ledger_accounts
        WHERE id = $1
    `

	var balance numeric.BigDecimal
	err = tx.QueryRow(ctx, query, r.LedgerAccountID).Scan(&balance)
	if err != nil {
		return nil, fmt.Errorf("failed to query user balance: %w", err)
	}

	if balance.Cmp(&r.BetAmount.Big) < 0 {
		return nil, finance.ErrInsufficientFunds
	}

	// Recompute gain for the user
	// Validate it's equal to the provided BetRequest
	possible, actualGain, feePaid, avgPrice, err := PossibleGainFeePriceForBuy(qVec, idx, m.Alpha, m.Fee, r.BetAmount, m.CapPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to recompute actual gain: %w", err)
	}

	if !possible {
		return nil, ErrInvalidBetAmount
	}

	if actualGain.Cmp(&r.MinWantedGain.Big) < 0 {
		fmt.Println("invalid quoted gain")
		fmt.Println("actual gain", actualGain.String(), "min wanted gain", r.MinWantedGain.String())
		return nil, ErrInvalidQuotedGain
	}

	// Everything is valid, we can start comiting

	// Make transaction
	transferID, err := finance.TransferMoney(ctx, tx, r.LedgerAccountID, m.HouseLedgerAccountID, *r.BetAmount, r.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer: %w", err)
	}

	// Commit a bet, verify time with close_time > now()
	bet := &Bet{
		LedgerAccountID:  r.LedgerAccountID,
		LedgerTransferID: transferID,
		OutcomeID:        r.OutcomeID,
		Payout:           actualGain,
		TotalPricePaid:   r.BetAmount,
		FeeApplied:       m.Fee,
		FeePaid:          feePaid,
		AvgPrice:         avgPrice,
		IdempotencyKey:   r.IdempotencyKey,
	}

	fmt.Println("feeApplied", bet.FeeApplied.String())

	query = `
	INSERT INTO bets(ledger_account_id, ledger_transfer_id, outcome_id, payout, total_price_paid, fee_applied, fee_paid, avg_price, idempotency_key)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
	ON CONFLICT (idempotency_key) DO NOTHING
	RETURNING id, placed_at
	`

	err = tx.QueryRow(ctx, query, bet.LedgerAccountID, bet.LedgerTransferID, bet.OutcomeID, bet.Payout, bet.TotalPricePaid, bet.FeeApplied, bet.FeePaid, bet.AvgPrice, bet.IdempotencyKey).Scan(&bet.ID, &bet.PlacedAt)

	// Already have a bet with this idempotency key
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBetAlreadyPlaced
	}

	if err != nil {
		return nil, fmt.Errorf("failed to insert bet: %w", err)
	}

	// Add the shares to the outcome
	if _, err := tx.Exec(ctx, `UPDATE outcomes SET quantity = quantity + $1, volume = volume + $2 WHERE id = $3`, actualGain, r.BetAmount, r.OutcomeID); err != nil {
		return nil, fmt.Errorf("failed to update outcome: %w", err)
	}

	// Update the market's version
	cmd, err := tx.Exec(ctx, `UPDATE markets SET version = version + 1, volume = volume + $1 WHERE id=$2 AND (close_time IS NULL OR close_time > now())`, r.BetAmount, r.MarketID)

	if err != nil {
		return nil, fmt.Errorf("failed to update market version: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return nil, ErrMarketNotFound
	}

	// Update the user's total wagered amount
	query = `UPDATE users
	SET total_wagered = total_wagered + $1
	WHERE id = $2`

	_, err = tx.Exec(ctx, query, r.BetAmount, r.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to update user's total wagered: %w", err)
	}

	return bet, tx.Commit(ctx)

}

type CashoutRequest = struct {
	BetID          uuid.UUID
	UserID         uuid.UUID
	MinWantedGain  *numeric.BigDecimal
	IdempotencyKey string
}

func (tm *TransactionManager) CashoutBet(ctx context.Context, r *CashoutRequest) (*BetCashout, error) {

	// Retrieve the cashed out bet
	// Check if wasn't already cashed out
	query := `
	SELECT m.id, b.id, b.ledger_account_id, b.ledger_transfer_id, b.outcome_id, b.payout, b.total_price_paid, b.fee_applied, b.fee_paid, b.avg_price, b.placed_at, b.idempotency_key
	FROM bets b
	JOIN outcomes o ON b.outcome_id = o.id
	JOIN markets m ON o.market_id = m.id
	JOIN ledger_accounts la ON b.ledger_account_id = la.id
	JOIN users u ON la.user_id = u.id
	WHERE b.id = $1 AND u.id = $2 AND NOT EXISTS (SELECT 1 FROM bet_cashouts bc WHERE bc.bet_id = b.id)
	`

	cashedBet := &Bet{}
	var marketID uuid.UUID

	err := tm.db.QueryRow(ctx, query, r.BetID, r.UserID).Scan(
		&marketID,
		&cashedBet.ID, &cashedBet.LedgerAccountID, &cashedBet.LedgerTransferID, &cashedBet.OutcomeID, &cashedBet.Payout, &cashedBet.TotalPricePaid,
		&cashedBet.FeeApplied, &cashedBet.FeePaid, &cashedBet.AvgPrice, &cashedBet.PlacedAt, &cashedBet.IdempotencyKey)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBetNotFound
		}
		return nil, fmt.Errorf("failed to retrieve bet: %w", err)
	}

	for attempt := range maxAttempts {

		betCashout, err := tm.cashoutBetOnce(ctx, marketID, cashedBet, r)
		if err == nil {
			// Push market update to redis
			go func() {
				if err := tm.PulishUpdateMarket(marketID); err != nil {
					tm.logger.Error("failed to publish market update", "error", err)
				}

				if err := tm.PublishBalanceUpdate(cashedBet.LedgerAccountID); err != nil {
					tm.logger.Error("failed to publish balance update", "error", err)
				}

				if err := tm.PublishBetUpdate(betCashout.ID); err != nil {
					tm.logger.Error("failed to publish bet update", "error", err)
				}

			}()
			return betCashout, nil
		}

		var pgErr *pgconn.PgError
		isRetryable := errors.As(err, &pgErr) && (pgErr.Code == pgerrcode.SerializationFailure || pgErr.Code == pgerrcode.DeadlockDetected)

		if !isRetryable {
			return nil, err
		}

		if attempt < maxAttempts-1 {
			select {
			// Larger backoff at each retry
			case <-time.After(time.Duration(50*(attempt+1)) * time.Millisecond):
			// Context is done, exit early
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		return nil, err
	}

	return nil, errors.New("too many serialization retries")
}

func (tm *TransactionManager) cashoutBetOnce(ctx context.Context, marketID uuid.UUID, cashedBet *Bet, r *CashoutRequest) (*BetCashout, error) {

	opts := pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	}

	tx, err := tm.db.BeginTx(ctx, opts)

	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}

	defer tx.Rollback(ctx)

	var m Market
	var outcomesIDs []int64
	var qVec []*numeric.BigDecimal

	query := `
	SELECT m.id, m.house_ledger_account_id, m.alpha, m.fee, m.cap_price,
	m.version,
	array_agg(o.quantity ORDER BY o.id) AS q_vec,
	array_agg(o.id ORDER BY o.id) AS outcome_ids
	FROM markets m
	JOIN outcomes o ON o.market_id = m.id
	WHERE m.id = $1 AND m.status = 'opened' AND (m.close_time IS NULL OR m.close_time > now())
	GROUP BY m.id`

	if err = tx.QueryRow(ctx, query, marketID).Scan(&m.ID, &m.HouseLedgerAccountID, &m.Alpha, &m.Fee, &m.CapPrice, &m.Version, &qVec, &outcomesIDs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMarketNotFound
		}
		return nil, fmt.Errorf("failed to query market's current state: %w", err)
	}

	if len(qVec) != len(outcomesIDs) || len(qVec) == 0 {
		return nil, errors.New("inconsistent outcomes for market")
	}

	// Find the index of the outcome in the q vector
	idx := slices.Index(outcomesIDs, cashedBet.OutcomeID)
	if idx == -1 {
		return nil, ErrOutcomeNotFound
	}

	// Recompute cashout gain for the user
	// cashedBet.Payout = number of shares bought
	fmt.Println("CAP PRICE", m.CapPrice)
	possible, cashoutGain, err := PossibleGainForSell(qVec, idx, m.Alpha, cashedBet.Payout, m.CapPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to recompute cashout gain: %w", err)
	}

	if !possible {
		return nil, ErrInvalidBetAmount
	}

	fmt.Println("CASHOUT GAIN:", cashoutGain.String())

	if cashoutGain.Cmp(&r.MinWantedGain.Big) < 0 || cashoutGain.Sign() <= 0 {
		return nil, ErrInvalidQuotedGain
	}

	transferID, err := finance.TransferMoney(ctx, tx, m.HouseLedgerAccountID, cashedBet.LedgerAccountID, *cashoutGain, r.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer: %w", err)
	}

	// Commit a bet cashout
	betCashout := &BetCashout{
		BetID:            cashedBet.ID,
		LedgerTransferID: transferID,
		Payout:           cashoutGain,
		IdempotencyKey:   r.IdempotencyKey,
	}

	query = `
	INSERT INTO bet_cashouts(bet_id, ledger_transfer_id, payout, idempotency_key)
	VALUES($1, $2, $3, $4)
	ON CONFLICT (idempotency_key) DO NOTHING
	RETURNING id
	`

	err = tx.QueryRow(ctx, query, betCashout.BetID, betCashout.LedgerTransferID, betCashout.Payout, betCashout.IdempotencyKey).Scan(&betCashout.ID)

	// Already have a cashout with this idempotency key
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBetAlreadyCashedOut
	}

	if err != nil {
		return nil, fmt.Errorf("failed to insert bet cashout: %w", err)
	}

	// Hedge the outcomes
	// cashedBet.Payout = number of shares bought = hedged number of shares
	query = `UPDATE outcomes 
	SET quantity = quantity + $1
	WHERE market_id = $2 AND id <> $3`

	_, err = tx.Exec(ctx, query, cashedBet.Payout, m.ID, cashedBet.OutcomeID)
	if err != nil {
		return nil, fmt.Errorf("failed to update outcomes: %w", err)
	}

	// Update the market's version
	cmd, err := tx.Exec(ctx, `UPDATE markets SET version = version + 1 WHERE id=$1 AND (close_time IS NULL OR close_time > now())`, m.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to update market version: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return nil, ErrMarketNotFound
	}

	return betCashout, tx.Commit(ctx)

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

	var houseAccountID uuid.UUID
	query := `SELECT house_ledger_account_id FROM markets
	WHERE id = $1 AND status IN ('opened','paused')
	`

	err = tx.QueryRow(ctx, query, marketID).Scan(&houseAccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMarketNotFound
		}
		return fmt.Errorf("failed to get house ledger account id: %w", err)
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
	SELECT b.ledger_account_id, SUM(b.payout)	
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

	type payoutInfo struct {
		ledgerAccountID uuid.UUID
		payout          numeric.BigDecimal
	}

	var payouts []payoutInfo

	for rows.Next() {
		var payout payoutInfo

		err = rows.Scan(&payout.ledgerAccountID, &payout.payout)
		if err != nil {
			return fmt.Errorf("failed to scan: %w", err)
		}
		payouts = append(payouts, payout)
	}

	rows.Close()

	for _, p := range payouts {
		idemKey := fmt.Sprintf("settle:%s:%s", marketID, p.ledgerAccountID)
		_, err = finance.TransferMoney(ctx, tx, houseAccountID, p.ledgerAccountID, p.payout, idemKey)
		if err != nil {
			return fmt.Errorf("failed to credit account %s: %w", p.ledgerAccountID, err)
		}
	}

	if rows.Err() != nil {
		return fmt.Errorf("error transactions bets rows: %w", rows.Err())
	}

	query = `UPDATE markets 
	SET status = 'resolved', version=version+1 
	WHERE id=$1`

	_, err = tx.Exec(ctx, query, marketID)
	if err != nil {
		return fmt.Errorf("failed to finalize settlement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	go func(marketID uuid.UUID) {
		if err := tm.PublishMarketResolved(winningOutcomeID); err != nil {
			tm.logger.Error("failed to publish market resolved", "error", err)
		}
	}(marketID)

	return nil
}

func (tm *TransactionManager) CancelMarket(ctx context.Context, marketID uuid.UUID) error {
	tx, err := tm.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	defer tx.Rollback(ctx)

	var houseAccountID uuid.UUID
	query := `SELECT house_ledger_account_id FROM markets
	WHERE id = $1 AND status IN ('opened','paused')
	`

	err = tx.QueryRow(ctx, query, marketID).Scan(&houseAccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMarketNotFound
		}
		return fmt.Errorf("failed to get house ledger account id: %w", err)
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
	SELECT b.ledger_account_id, SUM(b.total_price_paid) as refund	
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
		var refund numeric.BigDecimal

		err = rows.Scan(&ledgerAccountID, &refund)
		if err != nil {
			return fmt.Errorf("failed to scan")
		}

		idemKey := fmt.Sprintf("settle:%s:%s", marketID, ledgerAccountID)
		_, err = finance.TransferMoney(ctx, tx, houseAccountID, ledgerAccountID, refund, idemKey)
		if err != nil {
			return fmt.Errorf("failed to refund account %s: %w", ledgerAccountID, err)
		}
	}

	if rows.Err() != nil {
		return fmt.Errorf("error iterating bets rows: %w", rows.Err())
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

func (tm *TransactionManager) PulishUpdateMarket(marketID uuid.UUID) error {

	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mu := ps.MarketUpdate{
		MarketID: marketID,
	}

	buf, err := json.Marshal(mu)

	if err != nil {
		return fmt.Errorf("failed to marshal market update: %w", err)
	}

	return tm.rdb.Publish(updateCtx, ps.MarketUpdateChannel, string(buf)).Err()

}

func (tm *TransactionManager) PublishBetUpdate(betID uuid.UUID) error {

	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bu := ps.BetUpdate{
		BetID: betID,
	}

	buf, err := json.Marshal(bu)

	if err != nil {
		return fmt.Errorf("failed to marshal bet update: %w", err)
	}

	return tm.rdb.Publish(updateCtx, ps.BetUpdateChannel, string(buf)).Err()

}

func (tm *TransactionManager) PublishBalanceUpdate(ledgerAccountID uuid.UUID) error {

	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bu := ps.BalanceUpdate{
		LedgerAccountID: ledgerAccountID,
	}

	buf, err := json.Marshal(bu)
	if err != nil {
		return fmt.Errorf("failed to marshal balance update: %w", err)
	}

	return tm.rdb.Publish(updateCtx, "balance:update", string(buf)).Err()
}

func (tm *TransactionManager) PublishMarketResolved(winningOutcomeID int64) error {

	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mu := ps.MarketResolvedUpdate{
		WinningOutcomeID: winningOutcomeID,
	}

	buf, err := json.Marshal(mu)

	if err != nil {
		return fmt.Errorf("failed to marshal market update: %w", err)
	}

	return tm.rdb.Publish(updateCtx, ps.MarketResolvedChannel, string(buf)).Err()

}
