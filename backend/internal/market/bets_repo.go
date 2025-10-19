package market

import (
	"context"
	"fmt"
	"seer/internal/repos"
	"seer/internal/utils/meta"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BetManager struct {
	db *pgxpool.Pool
}

func NewBetManager(db *pgxpool.Pool) *BetManager {
	return &BetManager{
		db: db,
	}
}

func (bm *BetManager) SearchBets(ctx context.Context, bsq *BetSearchQuery) ([]BetView, *meta.Metadata, error) {

	query := fmt.Sprintf(`WITH bets_with_status AS (SELECT
            b.id AS id, 
            b.ledger_account_id AS ledger_account_id,
            u.id AS user_id, u.username AS username, u.hidden AS hidden,
            la.currency as currency,
            b.payout AS payout, b.total_price_paid AS total_price_paid, b.fee_applied AS fee_applied, b.fee_paid AS fee_paid, b.avg_price AS avg_price,
            b.placed_at AS placed_at,
            m.id AS market_id, m.name AS market_name,
            o.id AS outcome_id, o.name AS outcome_name, 
            m.status AS market_status,
            CASE
                WHEN m.status IN ('opened','paused','settling') THEN 'active'
                WHEN m.status = 'resolved'  AND mr.winning_outcome_id = b.outcome_id THEN 'won'
                WHEN m.status = 'resolved'  AND (mr.winning_outcome_id <> b.outcome_id) THEN 'lost'
                WHEN m.status = 'cancelled' THEN 'refunded'
                ELSE 'unknown'
            END AS bet_status
        FROM bets b
        JOIN ledger_accounts la ON b.ledger_account_id = la.id
        JOIN outcomes o ON b.outcome_id = o.id
        JOIN markets m ON o.market_id = m.id
        JOIN users u ON la.user_id = u.id
        LEFT JOIN market_resolutions mr ON mr.market_id = m.id
    )
    SELECT count(*) OVER() AS total_count, 
        id, 
        ledger_account_id,
        user_id, username, hidden,
        currency,
        payout, total_price_paid, fee_applied, fee_paid, avg_price,
        placed_at,
        market_id, market_name,
        outcome_id, outcome_name, 
        market_status, bet_status
    FROM bets_with_status
    WHERE ($1::UUID IS NULL OR user_id = $1)
    AND ($2::UUID IS NULL OR market_id = $2)
    AND ($3::bigint IS NULL OR total_price_paid >= $3)
    AND ($4::bigint IS NULL OR total_price_paid <= $4)
    AND ($5::TEXT IS NULL OR 
        CASE 
            WHEN $5 = 'active' THEN bet_status = 'active'
            WHEN $5 = 'resolved' THEN (bet_status = 'won' OR bet_status = 'lost')
            WHEN $5 = 'won' THEN bet_status = 'won'
            WHEN $5 = 'lost' THEN bet_status = 'lost'
            WHEN $5 = 'refunded' THEN bet_status = 'refunded'
            ELSE true
        END
    )
    ORDER BY %s, id DESC
    LIMIT $6 OFFSET $7
    `, bsq.GetOrderBy())

	rows, err := bm.db.Query(ctx, query, bsq.UserID, bsq.MarketID, bsq.MinPrice, bsq.MaxPrice, bsq.Status, bsq.Limit(), bsq.Offset())

	if err != nil {
		return nil, nil, fmt.Errorf("failed to query rows bets: %w", err)
	}

	defer rows.Close()

	bets := []BetView{}
	var totalCount int64

	for rows.Next() {
		var b BetView
		var marketStatus string

		err := rows.Scan(&totalCount,
			&b.ID,
			&b.LedgerAccountID,
			&b.User.ID, &b.User.Username, &b.User.Hidden,
			&b.Currency,
			&b.Payout, &b.TotalPricePaid, &b.FeeApplied, &b.FeePaid, &b.AvgPrice,
			&b.PlacedAt,
			&b.MarketID, &b.MarketName,
			&b.OutcomeID, &b.OutcomeName,
			&marketStatus, &b.Status)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan bet: %w", err)
		}

		bets = append(bets, b)
	}

	if rows.Err() != nil {
		return nil, nil, fmt.Errorf("error iterating bets rows: %w", rows.Err())
	}

	metadata := meta.CalculateMetadata(totalCount, bsq.Page, bsq.PageSize)

	return bets, metadata, nil

}

func (bm *BetManager) GetBetView(ctx context.Context, betID uuid.UUID) (*BetView, error) {

	query := `SELECT b.id, 
    u.id, u.username, u.hidden, 
    b.payout, b.total_price_paid, b.fee_applied, b.fee_paid, b.avg_price,
    b.placed_at,
    m.id AS market_id, m.name AS market_name,
    o.id AS outcome_id, o.name AS outcome_name, 
    CASE
        WHEN m.status IN ('opened','paused','settling') THEN 'active'
        WHEN m.status = 'resolved'  AND mr.winning_outcome_id = b.outcome_id THEN 'won'
        WHEN m.status = 'resolved'  AND (mr.winning_outcome_id <> b.outcome_id) THEN 'lost'
        WHEN m.status = 'cancelled' THEN 'refunded'
        ELSE 'unknown'
    END AS bet_status
    FROM bets b
    JOIN ledger_accounts la ON b.ledger_account_id = la.id
    JOIN outcomes o ON b.outcome_id = o.id
    JOIN markets m ON o.market_id = m.id
    JOIN users u ON la.user_id = u.id
    LEFT JOIN market_resolutions mr ON mr.market_id = m.id
    WHERE b.id = $1
    `

	var b BetView
	err := bm.db.QueryRow(ctx, query, betID).Scan(&b.ID,
		&b.User.ID, &b.User.Username, &b.User.Hidden,
		&b.Payout, &b.TotalPricePaid, &b.FeeApplied, &b.FeePaid, &b.AvgPrice,
		&b.PlacedAt,
		&b.MarketID, &b.MarketName,
		&b.OutcomeID, &b.OutcomeName,
		&b.Status)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repos.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to query bet: %w", err)
	}

	return &b, nil
}
