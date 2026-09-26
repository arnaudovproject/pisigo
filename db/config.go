// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package db

import (
	"fmt"
	"net/url"
	"strings"
)

type SQLConfig struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	Params   map[string]string
	DSN      string
}

func (c SQLConfig) BuildDSN() (string, error) {
	if c.DSN != "" {
		return c.DSN, nil
	}
	switch strings.ToLower(c.Driver) {
	case "postgres", "postgresql", "pgx":
		return buildPostgresDSN(c), nil
	case "mysql":
		return buildMySQLDSN(c), nil
	case "sqlite", "sqlite3":
		if c.Database == "" {
			return "", ErrInvalidDSN
		}
		return c.Database, nil
	case "sqlserver", "mssql":
		return buildMSSQLDSN(c), nil
	default:
		return "", fmt.Errorf("db: unsupported driver %q", c.Driver)
	}
}

func buildPostgresDSN(c SQLConfig) string {
	host := c.Host
	if host == "" {
		host = "localhost"
	}
	port := c.Port
	if port == 0 {
		port = 5432
	}
	ssl := c.SSLMode
	if ssl == "" {
		ssl = "require"
	}
	u := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", host, port),
		Path:   c.Database,
	}
	if c.User != "" {
		if c.Password != "" {
			u.User = url.UserPassword(c.User, c.Password)
		} else {
			u.User = url.User(c.User)
		}
	}
	q := url.Values{}
	q.Set("sslmode", ssl)
	for k, v := range c.Params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func buildMySQLDSN(c SQLConfig) string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == 0 {
		port = 3306
	}
	user := c.User
	pass := c.Password
	params := url.Values{}
	params.Set("parseTime", "true")
	params.Set("charset", "utf8mb4")
	for k, v := range c.Params {
		params.Set(k, v)
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", user, pass, host, port, c.Database, params.Encode())
}

func buildMSSQLDSN(c SQLConfig) string {
	host := c.Host
	if host == "" {
		host = "localhost"
	}
	port := c.Port
	if port == 0 {
		port = 1433
	}
	q := url.Values{}
	q.Set("database", c.Database)
	if c.User != "" {
		q.Set("user id", c.User)
	}
	if c.Password != "" {
		q.Set("password", c.Password)
	}
	for k, v := range c.Params {
		q.Set(k, v)
	}
	return fmt.Sprintf("sqlserver://%s:%d?%s", host, port, q.Encode())
}
