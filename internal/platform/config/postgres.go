package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type PostgresConfig struct {
	DSN string
}

func LoadPostgresConfig() (PostgresConfig, error) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn != "" {
		if err := validateDSN(dsn); err != nil {
			return PostgresConfig{}, err
		}
		return PostgresConfig{DSN: dsn}, nil
	}

	host := strings.TrimSpace(os.Getenv("PGHOST"))
	port := strings.TrimSpace(os.Getenv("PGPORT"))
	db := strings.TrimSpace(os.Getenv("PGDATABASE"))
	user := strings.TrimSpace(os.Getenv("PGUSER"))
	pass := os.Getenv("PGPASSWORD")
	sslMode := strings.TrimSpace(os.Getenv("PGSSLMODE"))
	if sslMode == "" {
		sslMode = "disable"
	}

	if host == "" || db == "" || user == "" || pass == "" {
		return PostgresConfig{}, fmt.Errorf("postgres config missing: set PGHOST/PGPORT/PGDATABASE/PGUSER/PGPASSWORD or DATABASE_URL")
	}
	if port == "" {
		port = "5432"
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, pass),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   db,
	}
	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()

	dsn = u.String()
	if err := validateDSN(dsn); err != nil {
		return PostgresConfig{}, err
	}

	return PostgresConfig{DSN: dsn}, nil
}

func validateDSN(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("invalid postgres dsn: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("invalid postgres dsn scheme: %s", u.Scheme)
	}
	if strings.TrimSpace(u.Host) == "" {
		return fmt.Errorf("invalid postgres dsn host")
	}
	return nil
}
