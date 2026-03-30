package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type User struct {
	ID    int64
	Name  string
	Email string
}

func New(ctx context.Context, dsn string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func RunInitSQL(ctx context.Context, conn *sql.DB, path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read init sql: %w", err)
	}

	if _, err := conn.ExecContext(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("execute init sql: %w", err)
	}
	return nil
}

func ListUsers(ctx context.Context, conn *sql.DB) ([]User, error) {
	const q = `
SELECT id, name, email
FROM users
ORDER BY id ASC;
`
	rows, err := conn.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
