package handlers

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"accountingweb/crypto"
	"accountingweb/db"
	appmiddleware "accountingweb/middleware"
)

var incomeStatementTmpl = template.Must(template.ParseFiles("templates/base.html", "templates/income_statement.html"))

type incomeStatementPageData struct {
	Username         string
	From             string
	To               string
	RevenueRows      []incomeRow
	ExpenseRows      []incomeRow
	TotalRevenue     string
	TotalExpenses    string
	NetIncome        string
	NetIncomePercent string
}

type incomeRow struct {
	AccountName string
	Total       string
	Percent     string
}

func IncomeStatementGet(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	dekB64 := appmiddleware.GetDEK(r)

	dek, err := base64.StdEncoding.DecodeString(dekB64)
	if err != nil {
		log.Printf("income statement: decode dek: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		earliest, found, err := db.GetEarliestTransactionDate(r.Context(), userID)
		if err != nil {
			log.Printf("income statement: get earliest date: %v", err)
			http.Error(w, "Failed to load date range", http.StatusInternalServerError)
			return
		}
		if found {
			from = earliest.Format("2006-01-02")
		} else {
			from = time.Now().Format("2006-01-02")
		}
		to = time.Now().Format("2006-01-02")
	}

	txRows, err := db.GetTransactionsForIncomeStatement(r.Context(), userID, from, to)
	if err != nil {
		log.Printf("income statement: fetch transactions: %v", err)
		http.Error(w, "Failed to load transactions", http.StatusInternalServerError)
		return
	}

	revenueTotals := map[string]float64{}
	expenseTotals := map[string]float64{}

	for _, tx := range txRows {
		amountStr, err := crypto.DecryptField(dek, tx.AmountCT, tx.AmountIV)
		if err != nil {
			log.Printf("income statement: decrypt amount: %v", err)
			http.Error(w, "Failed to decrypt transaction data", http.StatusInternalServerError)
			return
		}
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			log.Printf("income statement: parse amount: %v", err)
			http.Error(w, "Failed to parse transaction data", http.StatusInternalServerError)
			return
		}

		if tx.CreditAccountType == "revenue" {
			revenueTotals[tx.CreditAccountName] += amount
		}
		if tx.DebitAccountType == "expense" {
			expenseTotals[tx.DebitAccountName] += amount
		}
	}

	revenueRows, totalRevenue := toSortedRows(revenueTotals)
	expenseRows, totalExpenses := toSortedRows(expenseTotals)

	data := incomeStatementPageData{
		Username:         appmiddleware.GetUsername(r),
		From:             from,
		To:               to,
		RevenueRows:      revenueRows,
		ExpenseRows:      expenseRows,
		TotalRevenue:     fmt.Sprintf("%.2f", totalRevenue),
		TotalExpenses:    fmt.Sprintf("%.2f", totalExpenses),
		NetIncome:        fmt.Sprintf("%.2f", totalRevenue-totalExpenses),
		NetIncomePercent: formatPercent(totalRevenue-totalExpenses, totalRevenue),
	}

	if r.Header.Get("HX-Request") == "true" {
		incomeStatementTmpl.ExecuteTemplate(w, "results", data)
		return
	}

	incomeStatementTmpl.ExecuteTemplate(w, "base", data)
}

func toSortedRows(totals map[string]float64) ([]incomeRow, float64) {
	names := make([]string, 0, len(totals))
	var total float64
	for name, v := range totals {
		names = append(names, name)
		total += v
	}
	sort.Slice(names, func(i, j int) bool {
		return totals[names[i]] > totals[names[j]]
	})

	var rows []incomeRow
	for _, name := range names {
		rows = append(rows, incomeRow{
			AccountName: name,
			Total:       fmt.Sprintf("%.2f", totals[name]),
			Percent:     formatPercent(totals[name], total),
		})
	}
	return rows, total
}

func formatPercent(part, total float64) string {
	if total == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", part/total*100)
}
