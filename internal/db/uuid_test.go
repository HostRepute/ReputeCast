package db

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// UUIDs, alone and in arrays, are written and read as their 16 bytes, not
// through uuid.UUID's driver.Valuer and sql.Scanner.
func TestUUIDCodecMovesBytes(t *testing.T) {
	m := pgtype.NewMap()
	registerUUID(m)
	u := uuid.MustParse("01a17e44-8943-727f-a008-8d8dd7fc20e3")

	if _, ok := m.PlanEncode(pgtype.UUIDOID, pgtype.BinaryFormatCode, u).(encodeUUIDPlan); !ok {
		t.Fatalf("a uuid.UUID isn't encoded as bytes")
	}
	buf, err := m.Encode(pgtype.UUIDOID, pgtype.BinaryFormatCode, u, nil)
	if err != nil || string(buf) != string(u[:]) {
		t.Fatalf("encoded %x %v", buf, err)
	}
	var got uuid.UUID
	if _, ok := m.PlanScan(pgtype.UUIDOID, pgtype.BinaryFormatCode, &got).(scanUUIDPlan); !ok {
		t.Fatalf("a uuid.UUID isn't scanned from bytes")
	}
	if err := m.Scan(pgtype.UUIDOID, pgtype.BinaryFormatCode, buf, &got); err != nil || got != u {
		t.Fatalf("scanned %v %v", got, err)
	}

	ids := []uuid.UUID{u, uuid.Nil, uuid.MustParse("01a17e44-894a-76ad-9f74-4dc92847313b")}
	arr, err := m.Encode(pgtype.UUIDArrayOID, pgtype.BinaryFormatCode, ids, nil)
	if err != nil {
		t.Fatal(err)
	}
	var back []uuid.UUID
	if err := m.Scan(pgtype.UUIDArrayOID, pgtype.BinaryFormatCode, arr, &back); err != nil || len(back) != 3 || back[0] != u || back[1] != uuid.Nil || back[2] != ids[2] {
		t.Fatalf("array round trip: %v %v", back, err)
	}

	// NULL: a pointer becomes nil, a value stays as it was.
	p := &u
	if err := m.Scan(pgtype.UUIDOID, pgtype.BinaryFormatCode, nil, &p); err != nil || p != nil {
		t.Fatalf("NULL into *uuid.UUID: %v %v", p, err)
	}
	kept := u
	if err := m.Scan(pgtype.UUIDOID, pgtype.BinaryFormatCode, nil, &kept); err != nil || kept != u {
		t.Fatalf("NULL into uuid.UUID: %v %v", kept, err)
	}
	if err := m.Scan(pgtype.UUIDOID, pgtype.BinaryFormatCode, []byte{1, 2, 3}, &got); err == nil {
		t.Fatalf("a short value scanned")
	}

	// The text format is pgx's own.
	txt, err := m.Encode(pgtype.UUIDOID, pgtype.TextFormatCode, u, nil)
	if err != nil || string(txt) != u.String() {
		t.Fatalf("text: %q %v", txt, err)
	}
	var fromText uuid.UUID
	if err := m.Scan(pgtype.UUIDOID, pgtype.TextFormatCode, []byte(u.String()), &fromText); err != nil || fromText != u {
		t.Fatalf("from text: %v %v", fromText, err)
	}
}
