package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"temp/internal/database"
	"temp/internal/token"
)

func main() {
	dsn := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/app_db?sslmode=disable")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db connection failed: %v", err)
	}
	defer db.Close()

	if err := database.RunInitSQL(ctx, db, "sql/init.sql"); err != nil {
		log.Fatalf("init sql failed: %v", err)
	}

	if err := database.RegisterUser(ctx, db, "kim", "kim@example.com", "secret123"); err != nil {
		log.Fatalf("register user failed: %v", err)
	}

	u, err := database.Login(ctx, db, "kim@example.com", "secret123")
	if err != nil {
		log.Fatalf("login failed: %v", err)
	}
	log.Printf("login ok: id=%d, name=%s, email=%s", u.ID, u.Name, u.Email)

	if err := database.ChangePassword(ctx, db, "kim@example.com", "secret123", "newSecret456"); err != nil {
		log.Fatalf("change password failed: %v", err)
	}
	log.Println("password changed")

	if _, err := database.Login(ctx, db, "kim@example.com", "secret123"); err == nil {
		log.Fatal("expected login to fail with old password")
	} else if !errors.Is(err, database.ErrInvalidCredentials) {
		log.Fatalf("unexpected login error with old password: %v", err)
	}
	u2, err := database.Login(ctx, db, "kim@example.com", "newSecret456")
	if err != nil {
		log.Fatalf("login with new password failed: %v", err)
	}
	log.Printf("login with new password ok: id=%d", u2.ID)

	jwtSecret := []byte(getEnv("JWT_SECRET", "dev-only-change-me-use-long-random-secret"))
	jwtStr, err := token.Sign(u2.ID, u2.Email, jwtSecret, 1*time.Hour)
	if err != nil {
		log.Fatalf("issue jwt failed: %v", err)
	}
	log.Printf("issued jwt (prefix): %.40s...", jwtStr)

	claims, err := token.Parse(jwtStr, jwtSecret)
	if err != nil {
		log.Fatalf("parse jwt failed: %v", err)
	}
	log.Printf("jwt verified: uid=%d email=%s exp=%v", claims.UserID, claims.Email, claims.ExpiresAt)

	users, err := database.ListUsers(ctx, db)
	if err != nil {
		log.Fatalf("list users failed: %v", err)
	}

	log.Println("connected and initialized successfully")
	for _, u := range users {
		log.Printf("user: id=%d, name=%s, email=%s", u.ID, u.Name, u.Email)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
