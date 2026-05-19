package middleware

import (
	"context"
	"net/http"
)

// contextKey is an unexported type for context keys in this package.
// This prevents collisions with keys from other packages.
type contextKey string

const contextKeyUserID   contextKey = "user_id"
const contextKeyUsername contextKey = "username"
const contextKeyDEK      contextKey = "dek"

// RequireAuth checks for a valid session. If found, it injects user info
// into the request context and calls the next handler.
// If not found, it redirects to /login.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := GetSession(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, ok := session.Values[SessionKeyUserID]
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		username, _ := session.Values[SessionKeyUsername].(string)
		dek, _ := session.Values[SessionKeyDEK].(string)

		// Inject values into request context so handlers can read them
		ctx := r.Context()
		ctx = context.WithValue(ctx, contextKeyUserID, userID)
		ctx = context.WithValue(ctx, contextKeyUsername, username)
		ctx = context.WithValue(ctx, contextKeyDEK, dek)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper functions for handlers to read from context cleanly

func GetUserID(r *http.Request) int {
	v, _ := r.Context().Value(contextKeyUserID).(int)
	return v
}

func GetUsername(r *http.Request) string {
	v, _ := r.Context().Value(contextKeyUsername).(string)
	return v
}

func GetDEK(r *http.Request) string {
	v, _ := r.Context().Value(contextKeyDEK).(string)
	return v
}