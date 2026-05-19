package models

type Account struct {
	ID            int
	Name          string
	AccountType   string
	DebitOrCredit string
	BalanceCT     []byte
	BalanceIV     []byte
	// DecryptedBalance is populated after decryption, never stored in DB
	DecryptedBalance string
}
