package models

import "time"

type User struct {
	ID           int
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type UserSecret struct {
	ID            int
	UserID        int
	KDF           string
	SaltB64       string
	DEKWrappedB64 string
	DEKIVB64      string
}