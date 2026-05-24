package handlers

import (
	"encoding/base64"
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

	http.Redirect(w, r, "/transaction_history", http.StatusSeeOther)
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
