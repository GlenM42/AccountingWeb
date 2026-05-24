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
	"accountingweb/models"
	appmiddleware "accountingweb/middleware"
)

var balanceSheetTmpl = template.Must(template.ParseFiles("templates/base.html", "templates/balance_sheet.html"))

type balanceSheetPageData struct {
	Username         string
	Assets           []balanceRow
	Liabilities      []balanceRow
	TotalAssets      string
	TotalLiabilities string
	TotalEquity      string
}

type balanceRow struct {
	Name    string
	Balance string
}

func decryptAndSum(dek []byte, accounts []models.Account) ([]balanceRow, float64, error) {
	var rows []balanceRow
	var total float64

	for _, a := range accounts {
		balance, err := crypto.DecryptField(dek, a.BalanceCT, a.BalanceIV)
		if err != nil {
			return nil, 0, fmt.Errorf("decrypt balance for account %q: %w", a.Name, err)
		}
		amount, err := strconv.ParseFloat(balance, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("parse balance for account %q: %w", a.Name, err)
		}
		total += amount
		rows = append(rows, balanceRow{Name: a.Name, Balance: balance})
	}

	return rows, total, nil
}

func BalanceSheetGet(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	dekB64 := appmiddleware.GetDEK(r)

	dek, err := base64.StdEncoding.DecodeString(dekB64)
	if err != nil {
		log.Printf("balance sheet: decode dek: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	assets, err := db.GetAccountsByType(r.Context(), userID, "asset")
	if err != nil {
		log.Printf("balance sheet: fetch assets: %v", err)
		http.Error(w, "Failed to load accounts", http.StatusInternalServerError)
		return
	}

	liabilities, err := db.GetAccountsByType(r.Context(), userID, "liability")
	if err != nil {
		log.Printf("balance sheet: fetch liabilities: %v", err)
		http.Error(w, "Failed to load accounts", http.StatusInternalServerError)
		return
	}

	assetRows, totalAssets, err := decryptAndSum(dek, assets)
	if err != nil {
		log.Printf("balance sheet: %v", err)
		http.Error(w, "Failed to decrypt account data", http.StatusInternalServerError)
		return
	}

	liabilityRows, totalLiabilities, err := decryptAndSum(dek, liabilities)
	if err != nil {
		log.Printf("balance sheet: %v", err)
		http.Error(w, "Failed to decrypt account data", http.StatusInternalServerError)
		return
	}

	balanceSheetTmpl.ExecuteTemplate(w, "base", balanceSheetPageData{
		Username:         appmiddleware.GetUsername(r),
		Assets:           assetRows,
		Liabilities:      liabilityRows,
		TotalAssets:      fmt.Sprintf("%.2f", totalAssets),
		TotalLiabilities: fmt.Sprintf("%.2f", totalLiabilities),
		TotalEquity:      fmt.Sprintf("%.2f", totalAssets-totalLiabilities),
	})
}
