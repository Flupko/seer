package finance

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInsufficientFunds = errors.New("not enough funds")
	ErrAccountNotFound   = errors.New("account not found")
	ErrSameAccount       = errors.New("from and to accounts must differ")
	ErrInvalidAmount     = errors.New("amount must be > 0")
	ErrIdempotency       = errors.New("idemptotency key already used")
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
	ID          uuid.UUID
	AccountType string
	Balance     int64
	Currency    string
}

func TransferMoney(ctx context.Context, tx pgx.Tx, fromAccountID uuid.UUID, toAccountID uuid.UUID, amountMinor int64, idempotencyKey string) error {

	if amountMinor <= 0 {
		return ErrInvalidAmount
	}

	if fromAccountID == toAccountID {
		return ErrSameAccount
	}

	if idempotencyKey == "" {
		return fmt.Errorf("missing idempotency key")
	}

	var accountFrom, accountTo Account

	// Acquire both locks at the same time avoid a deadlock (where x locks row A, y locks row B, x waits for
	// acquire the lock on row B, y waits to acquire the lock on row A)

	query := `
        SELECT id, currency, balance
        FROM ledger_accounts
        WHERE id = ANY($1::uuid[])
		ORDER BY id
        FOR UPDATE
    `

	rows, err := tx.Query(ctx, query, []uuid.UUID{fromAccountID, toAccountID})
	if err != nil {
		return fmt.Errorf("failed to lock accounts: %w", err)
	}

	defer rows.Close()

	nb := 0

	for rows.Next() {
		var acc Account
		if err := rows.Scan(&acc.ID, &acc.Currency, &acc.Balance); err != nil {
			return fmt.Errorf("failed to scan account: %w", err)
		}

		switch acc.ID {
		case fromAccountID:
			accountFrom = acc
		case toAccountID:
			accountTo = acc
		}

		nb++

	}

	if err := rows.Err(); err != nil {
		return err
	}

	if nb != 2 {
		return fmt.Errorf("some accounts were not found")
	}

	// Compare currencies
	if accountFrom.Currency != accountTo.Currency {
		return fmt.Errorf("source currency %s doesn't match destination currency %s", accountFrom.Currency, accountTo.Currency)
	}

	// Check enough funds
	if accountFrom.Balance < amountMinor {
		return ErrInsufficientFunds
	}

	// Insert into ledger

	transferID := uuid.New()

	query = `INSERT INTO ledger_transfers(id, from_account_id, to_account_id, amount_minor, idempotency_key) 
		VALUES($1, $2, $3, $4, $5)
		ON CONFLICT (idempotency_key) DO NOTHING`
	cmd, err := tx.Exec(ctx, query, transferID, fromAccountID, toAccountID, amountMinor, idempotencyKey)
	if err != nil {
		return fmt.Errorf("failed to insert transfer: %w", err)
	}

	// Idempotent
	if cmd.RowsAffected() == 0 {
		return ErrIdempotency
	}

	// Debit part
	query = `
    WITH acc AS (
        UPDATE ledger_accounts
        SET balance = balance - $1, version = version + 1
        WHERE id = $2 AND balance >= $1
        RETURNING id, balance AS new_balance, version AS new_version
    )
    INSERT INTO ledger_entries (
        account_id, transfer_id, amount_minor, 
        account_previous_balance, account_current_balance, account_version, 
        metadata
    ) 
    SELECT id, $3, -$1, acc.new_balance + $1, acc.new_balance, acc.new_version, $4 FROM acc
	`

	cmd, err = tx.Exec(ctx, query, amountMinor, fromAccountID, transferID, "{}")
	if err != nil {
		return fmt.Errorf("failed to debit account %s: %w", fromAccountID, err)
	}

	if cmd.RowsAffected() == 0 {
		// balance >= $1 check failed
		return ErrInsufficientFunds
	}

	// Credit part
	query = `
    WITH acc AS (
        UPDATE ledger_accounts
        SET balance = balance + $1, version = version + 1
        WHERE id = $2
        RETURNING id, balance AS new_balance, version AS new_version
    )
    INSERT INTO ledger_entries (
        account_id, transfer_id, amount_minor, 
        account_previous_balance, account_current_balance, account_version, 
        metadata
    ) 
    SELECT id, $3, $1, acc.new_balance - $1, acc.new_balance, acc.new_version, $4 FROM acc
	`

	_, err = tx.Exec(ctx, query, amountMinor, toAccountID, transferID, "{}")
	if err != nil {
		return fmt.Errorf("failed to credit account %s: %w", toAccountID, err)
	}

	return nil

}

func GetUserAccountForCurrency(ctx context.Context, tx pgx.Tx, userID uuid.UUID, currency string) (uuid.UUID, error) {
	var accountID uuid.UUID
	query := `SELECT id FROM ledger_accounts WHERE user_id = $1 AND currency = $2 AND account_type = 'user'`
	err := tx.QueryRow(ctx, query, userID, currency).Scan(&accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("no account found for user in currency %s", currency)
		}
		return uuid.Nil, err
	}

	return accountID, nil
}
