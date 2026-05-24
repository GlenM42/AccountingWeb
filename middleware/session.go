package middleware

import (
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

const SessionName = "accounting_session"

// Store is the global cookie store. The secret key signs cookies so they
// can't be tampered with client-side.
// We have to initialize it lazily b/c packages run before `main.go`, where
// we load the .env file.
var Store *sessions.CookieStore

func InitStore() {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		panic("SESSION_SECRET environment variable is not set")
	}
	Store = sessions.NewCookieStore([]byte(secret))
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
}

// Helper to get the session from a request without repeating error handling
func GetSession(r *http.Request) (*sessions.Session, error) {
	return Store.Get(r, SessionName)
}