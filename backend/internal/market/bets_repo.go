package market

import (
	"context"
	"fmt"

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

func (bm *BetManager) SearchBets(ctx context.Context, bsq *BetSearchQuery) ([]BetView, error) {

	query := `WITH bets_with_status AS (
        SELECT b.id, la.user_id, 
            b.payout_cents, b.total_price_paid_cents, b.fee_paid_cents, b.fee_ppm,
            b.purchase_time,
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
        LEFT JOIN market_resolutions mr ON mr.market_id = m.id
    )
    SELECT * FROM bets_with_status
    WHERE ($1::UUID IS NULL OR user_id = $1)
    AND ($2::UUID IS NULL OR market_id = $2)
    AND ($3::TEXT IS NULL OR 
        CASE 
            WHEN $3 = 'active' THEN bet_status = 'active'
            WHEN $3 = 'resolved' THEN (bet_status = 'won' OR bet_status = 'lost')
            WHEN $3 = 'won' THEN bet_status = 'won'
            WHEN $3 = 'lost' THEN bet_status = 'lost'
            WHEN $3 = 'refunded' THEN bet_status = 'refunded'
            ELSE true
        END
    )
    ORDER BY purchase_time DESC, id DESC
    LIMIT $4 OFFSET $5
    `

	rows, err := bm.db.Query(ctx, query, bsq.UserID, bsq.MarketID, bsq.Status, bsq.Limit(), bsq.Offset())

	if err != nil {
		return nil, fmt.Errorf("failed to query rows bets: %w", err)
	}

	defer rows.Close()

	bets := []BetView{}

	for rows.Next() {
		var b BetView
		var marketStatus string

		err := rows.Scan(&b.ID, &b.UserID,
			&b.PayoutCents, &b.TotalPricePaidCents, &b.FeePaidCents, &b.FeePPM,
			&b.PurchaseTime,
			&b.MarketID, &b.MarketName,
			&b.OutcomeID, &b.OutcomeName,
			&marketStatus, &b.Status)

		if err != nil {
			return nil, fmt.Errorf("failed to scan bet: %w", err)
		}

		bets = append(bets, b)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return bets, nil

}
