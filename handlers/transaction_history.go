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

var transactionHistoryTmpl = template.Must(template.ParseFiles("templates/base.html", "templates/transaction_history.html"))

type transactionHistoryPageData struct {
	Username     string
	Transactions []txRow
}

type txRow struct {
	Date        string
	Debit       string
	Credit      string
	Description string
	Amount      string
}

func TransactionHistoryGet(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	dekB64 := appmiddleware.GetDEK(r)

	dek, err := base64.StdEncoding.DecodeString(dekB64)
	if err != nil {
		log.Printf("transaction history: decode dek: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	transactions, err := db.GetTransactionsByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("transaction history: fetch transactions: %v", err)
		http.Error(w, "Failed to load transactions", http.StatusInternalServerError)
		return
	}

	var rows []txRow
	for _, t := range transactions {
		description, err := crypto.DecryptField(dek, t.DescriptionCT, t.DescriptionIV)
		if err != nil {
			log.Printf("transaction history: decrypt description for tx %d: %v", t.ID, err)
			http.Error(w, "Failed to decrypt transaction data", http.StatusInternalServerError)
			return
		}
		amount, err := crypto.DecryptField(dek, t.AmountCT, t.AmountIV)
		if err != nil {
			log.Printf("transaction history: decrypt amount for tx %d: %v", t.ID, err)
			http.Error(w, "Failed to decrypt transaction data", http.StatusInternalServerError)
			return
		}
		f, err := strconv.ParseFloat(amount, 64)
		if err != nil {
			log.Printf("transaction history: parse amount for tx %d: %v", t.ID, err)
			http.Error(w, "Failed to parse transaction data", http.StatusInternalServerError)
			return
		}
		rows = append(rows, txRow{
			Date:        t.TransactionDate.Format("2006-01-02"),
			Debit:       t.DebitAccount,
			Credit:      t.CreditAccount,
			Description: description,
			Amount:      fmt.Sprintf("%.2f", f),
		})
	}

	transactionHistoryTmpl.ExecuteTemplate(w, "base", transactionHistoryPageData{
		Username:     appmiddleware.GetUsername(r),
		Transactions: rows,
	})
}
