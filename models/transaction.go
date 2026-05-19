package models

import "time"

type IncomeStatementRow struct {
	DebitAccountName  string
	DebitAccountType  string
	CreditAccountName string
	CreditAccountType string
	AmountCT          []byte
	AmountIV          []byte
}

type Transaction struct {
	ID              int
	TransactionDate time.Time
	DebitAccount    string
	CreditAccount   string
	DescriptionCT   []byte
	DescriptionIV   []byte
	AmountCT        []byte
	AmountIV        []byte
}
