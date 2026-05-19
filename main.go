package main

import (
	"log"
	"net/http"

	"accountingweb/db"
	"accountingweb/handlers"
	appmiddleware "accountingweb/middleware"

	"github.com/joho/godotenv"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	godotenv.Load() // Load .env file into environment variables
	appmiddleware.InitStore() // now SESSION_SECRET is loaded

	// Ensure DB connection first
	if err := db.Init(); err != nil {
		// log.Fatalf will print and call `os.Exit(1)`
		log.Fatalf("DB init failed: %v\n", err)
	}
	defer db.Pool.Close() // closes all connections after `main` returns

	r := chi.NewRouter()

	// Middleware: logs every request to stdout, recovers from panics
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", handlers.LoginGet)
	r.Get("/dashboard", handlers.DashboardGet)
	r.Get("/login", handlers.LoginGet)
	r.Post("/login", handlers.LoginPost)
	r.Post("/logout", handlers.LogoutPost)

	// Routes protected by the middleware RequireAuth
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth)
		r.Get("/balance_sheet", handlers.BalanceSheetGet)
		r.Get("/transaction_history", handlers.TransactionHistoryGet)
		r.Get("/new_transaction", handlers.NewTransactionGet)
		r.Post("/new_transaction", handlers.NewTransactionPost)
		r.Get("/income_statement", handlers.IncomeStatementGet)
	})

	log.Println("Listening on http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
