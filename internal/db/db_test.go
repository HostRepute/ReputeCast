package db_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

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

// Through a real connection: UUIDs and their arrays round-trip, NULLs scan
// as before, and a UUID still fits where Postgres expects text, both ways.
func TestOpenUUIDsRoundTrip(t *testing.T) {
	ctx := context.Background()
	pool, err := db.Open(ctx, testdb.URL(testdb.New(t)), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	u := uuid.MustParse("01a17e44-8943-727f-a008-8d8dd7fc20e3")
	ids := []uuid.UUID{u, uuid.Nil}
	var one uuid.UUID
	var many []uuid.UUID
	var text string
	var null *uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT $1::uuid, $2::uuid[], $1::uuid::text, NULL::uuid", u, ids).Scan(&one, &many, &text, &null); err != nil {
		t.Fatal(err)
	}
	if one != u || len(many) != 2 || many[0] != u || many[1] != uuid.Nil || text != u.String() || null != nil {
		t.Fatalf("round trip: %v %v %q %v", one, many, text, null)
	}
	var fromText uuid.UUID
	var asText string
	if err := pool.QueryRow(ctx, "SELECT $1::text::uuid, $2::text", u.String(), u).Scan(&fromText, &asText); err != nil || fromText != u || asText != u.String() {
		t.Fatalf("text: %v %q %v", fromText, asText, err)
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM unnest($1::uuid[]) AS id WHERE id = ANY($2::uuid[])", ids, []uuid.UUID{u}).Scan(&n); err != nil || n != 1 {
		t.Fatalf("ANY: %d %v", n, err)
	}
}
