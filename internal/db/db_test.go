package db_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/testdb"
)

func TestOpenTurnsJITOff(t *testing.T) {
	ctx := context.Background()
	pool, err := db.Open(ctx, testdb.URL(testdb.New(t)), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var jit string
	if err := pool.QueryRow(ctx, "SHOW jit").Scan(&jit); err != nil {
		t.Fatal(err)
	}
	if jit != "off" {
		t.Fatalf("jit = %q, want off", jit)
	}
}
