// Package testdb creates throwaway databases with the Asternic schema for
// integration tests. Tests using it are skipped unless PARSELOG_TEST_DSN
// points at a MySQL/MariaDB server, e.g.
//
//	PARSELOG_TEST_DSN='root:root@tcp(127.0.0.1:3306)/' go test ./...
package testdb

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
)

// New creates an empty database loaded with sql/qstats.sql from the
// repository and returns its DSN. The database is dropped when the test ends.
func New(t *testing.T) string {
	t.Helper()
	root := os.Getenv("PARSELOG_TEST_DSN")
	if root == "" {
		t.Skip("PARSELOG_TEST_DSN not set")
	}
	cfg, err := mysql.ParseDSN(root)
	if err != nil {
		t.Fatalf("PARSELOG_TEST_DSN: %v", err)
	}
	cfg.MultiStatements = true
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	name := "parselogd_test_" + hex.EncodeToString(b)

	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec("DROP DATABASE IF EXISTS " + name)
		_ = admin.Close()
	})
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatalf("create database: %v", err)
	}

	cfg.DBName = name
	dsn := cfg.FormatDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck // test helper
	if _, err := db.Exec(schema(t)); err != nil {
		t.Fatalf("load schema: %v", err)
	}
	cfg.MultiStatements = false
	return cfg.FormatDSN()
}

// schema returns sql/qstats.sql without its CREATE DATABASE / USE lines.
func schema(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", "sql", "qstats.sql"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		u := strings.ToUpper(strings.TrimSpace(l))
		if strings.HasPrefix(u, "CREATE DATABASE") || strings.HasPrefix(u, "USE ") {
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}
