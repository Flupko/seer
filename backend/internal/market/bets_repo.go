package market

import (
	"context"
	"fmt"
	"seer/internal/repos"
	"seer/internal/utils"

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

func (bm *BetManager) SearchBets(ctx context.Context, bsq *BetSearchQuery) ([]BetView, *utils.Metadata, error) {

	query := fmt.Sprintf(`WITH bets_with_status AS (SELECT
            b.id AS id, la.user_id AS user_id, 
            b.payout_cents AS payout_cents, b.total_price_paid_cents AS total_price_paid_cents, b.fee_paid_cents AS fee_paid_cents, b.fee_ppm AS fee_ppm,
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
        LEFT JOIN market_resolutions mr ON mr.market_id = m.id
    )
    SELECT count(*) OVER() AS total_count, 
        id, user_id, 
        payout_cents, total_price_paid_cents, fee_paid_cents, fee_ppm,
        placed_at,
        market_id, market_name,
        outcome_id, outcome_name, 
        market_status, bet_status
    FROM bets_with_status
    WHERE ($1::UUID IS NULL OR user_id = $1)
    AND ($2::UUID IS NULL OR market_id = $2)
    AND ($3::bigint IS NULL OR total_price_paid_cents >= $3)
    AND ($4::bigint IS NULL OR total_price_paid_cents <= $4)
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
    ORDER BY %s, id
    LIMIT $6 OFFSET $7
    `, bsq.GetOrderBy())

	rows, err := bm.db.Query(ctx, query, bsq.UserID, bsq.MarketID, bsq.MinPriceCents, bsq.MaxPriceCents, bsq.Status, bsq.Limit(), bsq.Offset())

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
			&b.ID, &b.UserID,
			&b.PayoutCents, &b.TotalPricePaidCents, &b.FeePaidCents, &b.FeePPM,
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

	metadata := utils.CalculateMetadata(totalCount, bsq.Page, bsq.PageSize)

	return bets, metadata, nil

}

func (bm *BetManager) GetBetView(ctx context.Context, betID uuid.UUID) (*BetView, error) {

	query := `SELECT b.id, la.user_id, 
    b.payout_cents, b.total_price_paid_cents, b.fee_paid_cents, b.fee_ppm,
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
    LEFT JOIN market_resolutions mr ON mr.market_id = m.id
    WHERE b.id = $1
    `

	var b BetView
	err := bm.db.QueryRow(ctx, query, betID).Scan(&b.ID, &b.UserID,
		&b.PayoutCents, &b.TotalPricePaidCents, &b.FeePaidCents, &b.FeePPM,
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
