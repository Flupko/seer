package finance

import (
	"context"
	"errors"
	"fmt"
	"seer/internal/numeric"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAccountNotFound      = errors.New("account not found")
	ErrCodeAcccountNotFound = "account_not_found"

	ErrLedgerAcountPolicy     = errors.New("ledger account policy violation")
	ErrCodeLedgerAcountPolicy = "ledger_account_policy"

	ErrSameAccount     = errors.New("from and to accounts must differ")
	ErrCodeSameAccount = "same_account"

	ErrInvalidAmount      = errors.New("amount must be > 0")
	ErrCodeInvalidAmmount = "invalid_amount"

	ErrIdempotency     = errors.New("idemptotency key already used")
	ErrCodeIdempotency = "idempotency_key"

	ErrDifferentCurrencies     = errors.New("cannot transfer between different currencies")
	ErrCodeDifferentCurrencies = "different_currencies"

	ErrInsufficientFunds = errors.New("insufficient funds")
)

type FinanceManager struct {
	db *pgxpool.Pool
}

func NewFinanceManager(db *pgxpool.Pool) *FinanceManager {
	return &FinanceManager{
		db: db,
	}
}

type LedgerAccountType string

const (
	AccountCustody         LedgerAccountType = "custody"
	AccountLiability       LedgerAccountType = "liability"
	AccountOwnerWithdrawal LedgerAccountType = "owner_withdrawal"
	AccountHouse           LedgerAccountType = "house"
)

type Currency string

const (
	USDT Currency = "USDT"
)

type LedgerEntry struct {
	ID                     uuid.UUID
	AccountID              uuid.UUID
	TransferID             uuid.UUID
	AmountMinor            int64
	AccountPreviousBalance int64
	AccountCurrentBalance  int64
	AccountVersion         int64
	Metadata               string
}

type Account struct {
	ID                   uuid.UUID
	AccountType          LedgerAccountType
	Balance              int64
	Currency             Currency
	AllowNegativeBalance bool
	AllowPostivieBalance bool
	CreatedAt            time.Time
	Version              int64
}

func TransferMoney(ctx context.Context, tx pgx.Tx, fromAccountID uuid.UUID, toAccountID uuid.UUID, amount numeric.BigDecimal, idempotencyKey string) (uuid.UUID, error) {

	if amount.Sign() <= 0 {
		return uuid.Nil, ErrInvalidAmount
	}

	if fromAccountID == toAccountID {
		return uuid.Nil, ErrSameAccount
	}

	if idempotencyKey == "" {
		return uuid.Nil, fmt.Errorf("missing idempotency key")
	}

	query := `SELECT ledger_create_transfer($1, $2, $3, $4)`

	var transferID uuid.UUID

	err := tx.QueryRow(ctx, query, fromAccountID, toAccountID, amount, idempotencyKey).Scan(&transferID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.RaiseException {
			switch {
			case strings.HasPrefix(pgErr.Message, ErrCodeLedgerAcountPolicy):
				return uuid.Nil, ErrLedgerAcountPolicy
			case strings.HasPrefix(pgErr.Message, ErrCodeSameAccount):
				return uuid.Nil, ErrSameAccount
			case strings.HasPrefix(pgErr.Message, ErrCodeInvalidAmmount):
				return uuid.Nil, ErrInvalidAmount
			case strings.HasPrefix(pgErr.Message, ErrCodeIdempotency):
				return uuid.Nil, ErrIdempotency
			case strings.HasPrefix(pgErr.Message, ErrCodeDifferentCurrencies):
				return uuid.Nil, ErrDifferentCurrencies
			case strings.HasPrefix(pgErr.Message, ErrCodeAcccountNotFound):
				return uuid.Nil, ErrAccountNotFound
			default:
				return uuid.Nil, fmt.Errorf("ledger transfer failed: %s", pgErr.Message)
			}

		}
		return uuid.Nil, err
	}

	return transferID, nil
}

func (fm *FinanceManager) GetLedgerAccountForCurrency(ctx context.Context, userID uuid.UUID, currency Currency, accountType LedgerAccountType) (uuid.UUID, error) {
	var accountID uuid.UUID

	query := `SELECT id FROM ledger_accounts WHERE user_id = $1 AND currency = $2 AND account_type = $3`
	err := fm.db.QueryRow(ctx, query, userID, currency, accountType).Scan(&accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("no account found for user in currency %s", currency)
		}
		return uuid.Nil, err
	}

	return accountID, nil
}

func (fm *FinanceManager) GetUserBalanceLiabiliy(ctx context.Context, userID uuid.UUID, cur Currency) (*numeric.BigDecimal, int64, error) {

	balance := &numeric.BigDecimal{}
	var version int64

	query := `SELECT balance, version FROM ledger_accounts WHERE user_id = $1 AND currency = $2 AND account_type = 'liability'`
	err := fm.db.QueryRow(ctx, query, userID, cur).Scan(&balance, &version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &numeric.BigDecimal{}, 0, ErrAccountNotFound
		}
		return &numeric.BigDecimal{}, 0, fmt.Errorf("failed to get user balance: %w", err)
	}
	return balance, version, nil
}

// func TransferMoney(ctx context.Context, tx pgx.Tx, fromAccountID uuid.UUID, toAccountID uuid.UUID, amountMinor int64, idempotencyKey string) error {

// 	if amountMinor <= 0 {
// 		return ErrInvalidAmount
// 	}

// 	if fromAccountID == toAccountID {
// 		return ErrSameAccount
// 	}

// 	if idempotencyKey == "" {
// 		return fmt.Errorf("missing idempotency key")
// 	}

// 	var accountFrom, accountTo Account

// 	// Acquire both locks at the same time avoid a deadlock (where x locks row A, y locks row B, x waits for
// 	// acquire the lock on row B, y waits to acquire the lock on row A)

// 	query := `
//         SELECT id, currency, balance
//         FROM ledger_accounts
//         WHERE id = ANY($1::uuid[])
// 		ORDER BY id
//         FOR UPDATE
//     `

// 	rows, err := tx.Query(ctx, query, []uuid.UUID{fromAccountID, toAccountID})
// 	if err != nil {
// 		return fmt.Errorf("failed to lock accounts: %w", err)
// 	}

// 	defer rows.Close()

// 	nb := 0

// 	for rows.Next() {
// 		var acc Account
// 		if err := rows.Scan(&acc.ID, &acc.Currency, &acc.Balance); err != nil {
// 			return fmt.Errorf("failed to scan account: %w", err)
// 		}

// 		switch acc.ID {
// 		case fromAccountID:
// 			accountFrom = acc
// 		case toAccountID:
// 			accountTo = acc
// 		}

// 		nb++

// 	}

// 	if nb != 2 {
// 		return fmt.Errorf("some accounts were not found")
// 	}

// 	// Compare currencies
// 	if accountFrom.Currency != accountTo.Currency {
// 		return fmt.Errorf("source currency %s doesn't match destination currency %s", accountFrom.Currency, accountTo.Currency)
// 	}

// 	// Check enough funds
// 	if accountFrom.Balance < amountMinor {
// 		return ErrInsufficientFunds
// 	}

// 	// Insert into ledger

// 	transferID := uuid.New()

// 	query = `INSERT INTO ledger_transfers(id, from_account_id, to_account_id, amount_minor, idempotency_key)
// 		VALUES($1, $2, $3, $4, $5)
// 		ON CONFLICT (idempotency_key) DO NOTHING`
// 	cmd, err := tx.Exec(ctx, query, transferID, fromAccountID, toAccountID, amountMinor, idempotencyKey)
// 	if err != nil {
// 		return fmt.Errorf("failed to insert transfer: %w", err)
// 	}

// 	// Idempotent
// 	if cmd.RowsAffected() == 0 {
// 		return ErrIdempotency
// 	}

// 	// Debit part
// 	query = `
//     WITH acc AS (
//         UPDATE ledger_accounts
//         SET balance = balance - $1, version = version + 1
//         WHERE id = $2 AND balance >= $1
//         RETURNING id, balance AS new_balance, version AS new_version
//     )
//     INSERT INTO ledger_entries (
//         account_id, transfer_id, amount_minor,
//         account_previous_balance, account_current_balance, account_version,
//         metadata
//     )
//     SELECT id, $3, -$1, acc.new_balance + $1, acc.new_balance, acc.new_version, $4 FROM acc
// 	`

// 	cmd, err = tx.Exec(ctx, query, amountMinor, fromAccountID, transferID, "{}")
// 	if err != nil {
// 		return fmt.Errorf("failed to debit account %s: %w", fromAccountID, err)
// 	}

// 	if cmd.RowsAffected() == 0 {
// 		// balance >= $1 check failed
// 		return ErrInsufficientFunds
// 	}

// 	// Credit part
// 	query = `
//     WITH acc AS (
//         UPDATE ledger_accounts
//         SET balance = balance + $1, version = version + 1
//         WHERE id = $2
//         RETURNING id, balance AS new_balance, version AS new_version
//     )
//     INSERT INTO ledger_entries (
//         account_id, transfer_id, amount_minor,
//         account_previous_balance, account_current_balance, account_version,
//         metadata
//     )
//     SELECT id, $3, $1, acc.new_balance - $1, acc.new_balance, acc.new_version, $4 FROM acc
// 	`

// 	_, err = tx.Exec(ctx, query, amountMinor, toAccountID, transferID, "{}")
// 	if err != nil {
// 		return fmt.Errorf("failed to credit account %s: %w", toAccountID, err)
// 	}

// 	return nil

// }
