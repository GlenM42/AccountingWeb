package db

import (
	"context"
	"fmt"

	"accountingweb/models"
)

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
