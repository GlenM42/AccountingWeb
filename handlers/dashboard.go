package handlers

import (
	"encoding/base64"
	"html/template"
	"net/http"

	"accountingweb/crypto"
	"accountingweb/db"
	appmiddleware "accountingweb/middleware"
)

var dashboardTmpl = template.Must(template.ParseFiles("templates/dashboard.html"))

type dashboardPageData struct {
	Username string
	Accounts []accountRow
}

type accountRow struct {
	Name          string
	AccountType   string
	DebitOrCredit string
	Balance       string
}

func DashboardGet(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	username := appmiddleware.GetUsername(r)
	dekB64 := appmiddleware.GetDEK(r)

	dek, err := base64.StdEncoding.DecodeString(dekB64)
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	accounts, err := db.GetAccountsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to load accounts", http.StatusInternalServerError)
		return
	}

	// Decrypt each account's balance before passing to template
	var rows []accountRow
	for _, a := range accounts {
		balance, err := crypto.DecryptField(dek, a.BalanceCT, a.BalanceIV)
		if err != nil {
			balance = "[decrypt error]"
		}
		rows = append(rows, accountRow{
			Name:          a.Name,
			AccountType:   a.AccountType,
			DebitOrCredit: a.DebitOrCredit,
			Balance:       balance,
		})
	}

	dashboardTmpl.Execute(w, dashboardPageData{
		Username: username,
		Accounts: rows,
	})
}
