package main

import (
	"fmt"
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

	if err := db.Init(); err != nil {
		log.Fatalf("DB init failed: %v\n", err)
	}

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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from accounting app!")
	})

	r.Get("/login", handlers.LoginGet)
	r.Post("/login", handlers.LoginPost)

	// Routes protected by the middleware RequireAuth
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth)
		r.Get("/dashboard", handlers.DashboardGet)
	})

	fmt.Println("Listening on http://localhost:8080")
	http.ListenAndServe(":8080", r)
}