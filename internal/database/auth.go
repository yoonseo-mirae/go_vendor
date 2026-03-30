package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

func hashPassword(plain string) (string, error) {
	if plain == "" {
		return "", fmt.Errorf("password must not be empty")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// RegisterUser creates or updates a user (by email) with a bcrypt password hash.
func RegisterUser(ctx context.Context, conn *sql.DB, name, email, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	const q = `
INSERT INTO users (name, email, password_hash)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE SET
	name = EXCLUDED.name,
	password_hash = EXCLUDED.password_hash;
`
	_, err = conn.ExecContext(ctx, q, name, email, hash)
	return err
}

// Login verifies email/password and returns the user (no password fields).
func Login(ctx context.Context, conn *sql.DB, email, password string) (User, error) {
	var u User
	var hash string
	const q = `SELECT id, name, email, password_hash FROM users WHERE email = $1`
	err := conn.QueryRowContext(ctx, q, email).Scan(&u.ID, &u.Name, &u.Email, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return u, nil
}

// ChangePassword verifies the current password and sets a new one.
func ChangePassword(ctx context.Context, conn *sql.DB, email, oldPassword, newPassword string) error {
	if _, err := Login(ctx, conn, email, oldPassword); err != nil {
		return err
	}
	newHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	const q = `UPDATE users SET password_hash = $1 WHERE email = $2`
	res, err := conn.ExecContext(ctx, q, newHash, email)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInvalidCredentials
	}
	return nil
}
