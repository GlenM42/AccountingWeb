package handlers

import (
	"log"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"accountingweb/crypto"
	"accountingweb/db"
	appmiddleware "accountingweb/middleware"

	"golang.org/x/crypto/pbkdf2"
)

var tmpl = template.Must(template.ParseFiles("templates/login.html"))

type loginPageData struct {
	Error string
}

func LoginGet(w http.ResponseWriter, r *http.Request) {
	tmpl.Execute(w, loginPageData{})
}

func LoginPost(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	// 1. Look up user
	user, err := db.GetUserByUsername(r.Context(), username)
	if err != nil {
		tmpl.Execute(w, loginPageData{Error: "Invalid credentials"})
		return
	}

	// 2. Verify password against stored Argon2id hash
	if !verifyPassword(password, user.PasswordHash) {
		tmpl.Execute(w, loginPageData{Error: "Invalid credentials"})
		return
	}

	// 3. Load user secrets and unwrap DEK
	secret, err := db.GetUserSecret(r.Context(), user.ID)
	if err != nil {
		tmpl.Execute(w, loginPageData{Error: "Invalid credentials"})
		return
	}

	kek, err := crypto.DeriveKEK(password, secret.SaltB64)
	if err != nil {
		tmpl.Execute(w, loginPageData{Error: "Invalid credentials"})
		return
	}

	dek, err := crypto.UnwrapDEK(kek, secret.DEKWrappedB64, secret.DEKIVB64)
	if err != nil {
		tmpl.Execute(w, loginPageData{Error: "Invalid credentials"})
		return
	}

	// 4. Store user info and DEK in session
	session, err := appmiddleware.GetSession(r)
	if err != nil {
		log.Printf("get session error: %v", err)
		tmpl.Execute(w, loginPageData{Error: "Session error"})
		return
	}

	session.Values[appmiddleware.SessionKeyUserID] = user.ID
	session.Values[appmiddleware.SessionKeyUsername] = user.Username
	session.Values[appmiddleware.SessionKeyDEK] = base64.StdEncoding.EncodeToString(dek)

	if err := session.Save(r, w); err != nil {
		log.Printf("session save error: %v", err)
		tmpl.Execute(w, loginPageData{Error: "Session error"})
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func LogoutPost(w http.ResponseWriter, r *http.Request) {
	session, err := appmiddleware.GetSession(r)
	if err == nil {
		session.Options.MaxAge = -1
		session.Save(r, w)
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// verifyPassword checks a plaintext password against a Django PBKDF2-SHA256 hash.
// Django format: pbkdf2_sha256$<iterations>$<salt>$<hash>
func verifyPassword(password, encoded string) bool {
    parts := strings.Split(encoded, "$")
    if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
        return false
    }

    iterations, err := strconv.Atoi(parts[1])
    if err != nil {
        return false
    }

    salt := parts[2]
    storedHash := parts[3]

    // Derive hash using same parameters Django used
    derived := pbkdf2.Key(
        []byte(password),
        []byte(salt),
        iterations,
        32, // SHA256 output is 32 bytes
        sha256.New,
    )

    derivedB64 := base64.StdEncoding.EncodeToString(derived)

    // subtle.ConstantTimeCompare prevents timing attacks —
    // comparing byte by byte in constant time so an attacker
    // can't infer correctness from how long the comparison took
    return subtle.ConstantTimeCompare([]byte(derivedB64), []byte(storedHash)) == 1
}