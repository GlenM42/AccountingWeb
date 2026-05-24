package handlers

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"accountingweb/crypto"
	"accountingweb/db"
	appmiddleware "accountingweb/middleware"
)

var newTransactionTmpl = template.Must(template.ParseFiles("templates/base.html", "templates/new_transaction.html"))

type newTransactionPageData struct {
	Username string
	Accounts []accountOption
	Error    string
}

type accountOption struct {
	ID   int
	Name string
}

func NewTransactionGet(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)

	accounts, err := db.GetAccountsByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("new transaction: fetch accounts: %v", err)
		http.Error(w, "Failed to load accounts", http.StatusInternalServerError)
		return
	}

	var options []accountOption
	for _, a := range accounts {
		options = append(options, accountOption{ID: a.ID, Name: a.Name})
	}

	newTransactionTmpl.ExecuteTemplate(w, "base", newTransactionPageData{
		Username: appmiddleware.GetUsername(r),
		Accounts: options,
	})
}

func NewTransactionPost(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	dekB64 := appmiddleware.GetDEK(r)

	dek, err := base64.StdEncoding.DecodeString(dekB64)
	if err != nil {
		log.Printf("new transaction: decode dek: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	debitID, err := strconv.Atoi(r.FormValue("debit_account_id"))
	if err != nil {
		renderNewTransactionError(w, r, userID, "Invalid debit account.")
		return
	}
	creditID, err := strconv.Atoi(r.FormValue("credit_account_id"))
	if err != nil {
		renderNewTransactionError(w, r, userID, "Invalid credit account.")
		return
	}
	date := r.FormValue("transaction_date")
	if date == "" {
		renderNewTransactionError(w, r, userID, "Transaction date is required.")
		return
	}
	description := r.FormValue("description")
	amount := r.FormValue("amount")
	if amount == "" {
		renderNewTransactionError(w, r, userID, "Amount is required.")
		return
	}
	if _, err := strconv.ParseFloat(amount, 64); err != nil {
		renderNewTransactionError(w, r, userID, "Amount must be a valid number.")
		return
	}

	descCT, descIV, err := crypto.EncryptField(dek, description)
	if err != nil {
		log.Printf("new transaction: encrypt description: %v", err)
		http.Error(w, "Encryption error", http.StatusInternalServerError)
		return
	}
	amountCT, amountIV, err := crypto.EncryptField(dek, amount)
	if err != nil {
		log.Printf("new transaction: encrypt amount: %v", err)
		http.Error(w, "Encryption error", http.StatusInternalServerError)
		return
	}

	if err := db.InsertTransaction(r.Context(), userID, debitID, creditID, date, descCT, descIV, amountCT, amountIV); err != nil {
		log.Printf("new transaction: insert: %v", err)
		http.Error(w, "Failed to save transaction", http.StatusInternalServerError)
		return
	}

	amt, _ := strconv.ParseFloat(amount, 64) // already validated above

	if err := applyBalanceUpdate(r, userID, dek, debitID, amt, true); err != nil {
		log.Printf("new transaction: update debit balance: %v", err)
		http.Error(w, "Failed to update account balance", http.StatusInternalServerError)
		return
	}
	if err := applyBalanceUpdate(r, userID, dek, creditID, amt, false); err != nil {
		log.Printf("new transaction: update credit balance: %v", err)
		http.Error(w, "Failed to update account balance", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/transaction_history", http.StatusSeeOther)
}

// applyBalanceUpdate fetches an account, adjusts its encrypted balance by amt,
// and writes it back. isDebit=true means this account is the debit side of the
// transaction; the sign depends on whether the account's normal balance side matches.
func applyBalanceUpdate(r *http.Request, userID int, dek []byte, accountID int, amt float64, isDebit bool) error {
	acct, err := db.GetAccountByID(r.Context(), userID, accountID)
	if err != nil {
		return fmt.Errorf("fetch account %d: %w", accountID, err)
	}

	balanceStr, err := crypto.DecryptField(dek, acct.BalanceCT, acct.BalanceIV)
	if err != nil {
		return fmt.Errorf("decrypt balance for account %d: %w", accountID, err)
	}

	current, err := strconv.ParseFloat(balanceStr, 64)
	if err != nil {
		return fmt.Errorf("parse balance for account %d: %w", accountID, err)
	}

	// A debit increases accounts whose normal side is debit (assets, expenses).
	// A credit increases accounts whose normal side is credit (liabilities, equity, revenue).
	// If the transaction side matches the account's normal side, add; otherwise subtract.
	normalIsDebit := acct.DebitOrCredit == "debit"
	if isDebit == normalIsDebit {
		current += amt
	} else {
		current -= amt
	}

	newBalanceStr := strconv.FormatFloat(current, 'f', 2, 64)
	ct, iv, err := crypto.EncryptField(dek, newBalanceStr)
	if err != nil {
		return fmt.Errorf("encrypt new balance for account %d: %w", accountID, err)
	}

	return db.UpdateAccountBalance(r.Context(), accountID, ct, iv)
}

func renderNewTransactionError(w http.ResponseWriter, r *http.Request, userID int, msg string) {
	accounts, err := db.GetAccountsByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("new transaction: fetch accounts for error page: %v", err)
		http.Error(w, "Failed to load accounts", http.StatusInternalServerError)
		return
	}

	var options []accountOption
	for _, a := range accounts {
		options = append(options, accountOption{ID: a.ID, Name: a.Name})
	}

	w.WriteHeader(http.StatusUnprocessableEntity)
	newTransactionTmpl.ExecuteTemplate(w, "base", newTransactionPageData{
		Username: appmiddleware.GetUsername(r),
		Accounts: options,
		Error:    msg,
	})
}
