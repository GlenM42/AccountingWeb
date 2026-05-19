package db

import (
	"context"
	"fmt"
	"time"

	"accountingweb/models"
)

func InsertTransaction(ctx context.Context, ownerID, debitAccountID, creditAccountID int, date string, descCT, descIV, amountCT, amountIV []byte) error {
	_, err := Pool.Exec(ctx, `
		INSERT INTO transactions
			(owner_id, debit_account_id, credit_account_id, transaction_date,
			 description_ct, description_iv, amount_ct, amount_iv)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, ownerID, debitAccountID, creditAccountID, date, descCT, descIV, amountCT, amountIV)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}

func GetEarliestTransactionDate(ctx context.Context, userID int) (time.Time, bool, error) {
	var t *time.Time
	err := Pool.QueryRow(ctx,
		"SELECT MIN(transaction_date) FROM transactions WHERE owner_id = $1",
		userID,
	).Scan(&t)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("get earliest transaction date: %w", err)
	}
	if t == nil {
		return time.Time{}, false, nil
	}
	return *t, true, nil
}

func GetTransactionsForIncomeStatement(ctx context.Context, userID int, from, to string) ([]models.IncomeStatementRow, error) {
	rows, err := Pool.Query(ctx, `
		SELECT
			da.name, da.account_type,
			ca.name, ca.account_type,
			t.amount_ct, t.amount_iv
		FROM transactions t
		JOIN accounts da ON da.id = t.debit_account_id
		JOIN accounts ca ON ca.id = t.credit_account_id
		WHERE t.owner_id = $1
		  AND t.transaction_date BETWEEN $2 AND $3
	`, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query income statement: %w", err)
	}
	defer rows.Close()

	var result []models.IncomeStatementRow
	for rows.Next() {
		var r models.IncomeStatementRow
		if err := rows.Scan(
			&r.DebitAccountName, &r.DebitAccountType,
			&r.CreditAccountName, &r.CreditAccountType,
			&r.AmountCT, &r.AmountIV,
		); err != nil {
			return nil, fmt.Errorf("scan income statement row: %w", err)
		}
		result = append(result, r)
	}
	return result, nil
}

func GetTransactionsByUserID(ctx context.Context, userID int) ([]models.Transaction, error) {
	rows, err := Pool.Query(ctx, `
		SELECT t.id, t.transaction_date,
		       da.name AS debit_account,
		       ca.name AS credit_account,
		       t.description_ct, t.description_iv,
		       t.amount_ct, t.amount_iv
		FROM transactions t
		JOIN accounts da ON da.id = t.debit_account_id
		JOIN accounts ca ON ca.id = t.credit_account_id
		WHERE t.owner_id = $1
		ORDER BY t.transaction_date DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(
			&t.ID, &t.TransactionDate,
			&t.DebitAccount, &t.CreditAccount,
			&t.DescriptionCT, &t.DescriptionIV,
			&t.AmountCT, &t.AmountIV,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, nil
}

func GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	row := Pool.QueryRow(ctx,
		"SELECT id, username, password_hash, created_at FROM users WHERE username = $1",
		username,
	)

	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func GetUserSecret(ctx context.Context, userID int) (*models.UserSecret, error) {
	row := Pool.QueryRow(ctx,
		"SELECT id, user_id, kdf, salt_b64, dek_wrapped_b64, dek_iv_b64 FROM user_secrets WHERE user_id = $1",
		userID,
	)

	var s models.UserSecret
	if err := row.Scan(&s.ID, &s.UserID, &s.KDF, &s.SaltB64, &s.DEKWrappedB64, &s.DEKIVB64); err != nil {
		return nil, fmt.Errorf("get user secret: %w", err)
	}
	return &s, nil
}

func GetAccountsByType(ctx context.Context, userID int, accountType string) ([]models.Account, error) {
	rows, err := Pool.Query(ctx, `
		SELECT a.id, a.name, a.account_type, a.debit_or_credit, a.balance_ct, a.balance_iv
		FROM accounts a
		JOIN account_users au ON au.account_id = a.id
		WHERE au.user_id = $1 AND a.account_type = $2
		ORDER BY a.name
	`, userID, accountType)
	if err != nil {
		return nil, fmt.Errorf("query accounts by type: %w", err)
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(&a.ID, &a.Name, &a.AccountType, &a.DebitOrCredit, &a.BalanceCT, &a.BalanceIV); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func GetAccountsByUserID(ctx context.Context, userID int) ([]models.Account, error) {
	rows, err := Pool.Query(ctx, `
		SELECT a.id, a.name, a.account_type, a.debit_or_credit, a.balance_ct, a.balance_iv
		FROM accounts a
		JOIN account_users au ON au.account_id = a.id
		WHERE au.user_id = $1
		ORDER BY a.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(&a.ID, &a.Name, &a.AccountType, &a.DebitOrCredit, &a.BalanceCT, &a.BalanceIV); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}
