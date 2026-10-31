package db

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// uuidCodec moves github.com/google/uuid values as the 16 bytes Postgres
// stores. uuid.UUID is a driver.Valuer and an sql.Scanner, which pgx
// otherwise honours by writing and parsing a 36-character string for each:
// sending the 31,000 contacts of an audience as a parameter took 38 ms of
// that. Everything else, text formats included, is pgx's own UUID codec.
type uuidCodec struct{ pgtype.UUIDCodec }

func (c uuidCodec) PlanEncode(m *pgtype.Map, oid uint32, format int16, value any) pgtype.EncodePlan {
	if _, ok := value.(uuid.UUID); ok && format == pgtype.BinaryFormatCode {
		return encodeUUIDPlan{}
	}
	return c.UUIDCodec.PlanEncode(m, oid, format, value)
}

func (c uuidCodec) PlanScan(m *pgtype.Map, oid uint32, format int16, target any) pgtype.ScanPlan {
	if _, ok := target.(*uuid.UUID); ok && format == pgtype.BinaryFormatCode {
		return scanUUIDPlan{}
	}
	return c.UUIDCodec.PlanScan(m, oid, format, target)
}

type encodeUUIDPlan struct{}

func (encodeUUIDPlan) Encode(value any, buf []byte) ([]byte, error) {
	u := value.(uuid.UUID)
	return append(buf, u[:]...), nil
}

type scanUUIDPlan struct{}

func (scanUUIDPlan) Scan(src []byte, dst any) error {
	if src == nil {
		// NULL leaves the UUID as it is, as uuid.UUID.Scan does.
		return nil
	}
	if len(src) != 16 {
		return fmt.Errorf("invalid length for UUID: %d", len(src))
	}
	copy(dst.(*uuid.UUID)[:], src)
	return nil
}

// registerUUID makes a connection's types use uuidCodec, for uuid and for
// uuid[], whose elements keep the codec they were registered with.
func registerUUID(m *pgtype.Map) {
	t := &pgtype.Type{Name: "uuid", OID: pgtype.UUIDOID, Codec: uuidCodec{}}
	m.RegisterType(t)
	m.RegisterType(&pgtype.Type{Name: "_uuid", OID: pgtype.UUIDArrayOID, Codec: &pgtype.ArrayCodec{ElementType: t}})
}
