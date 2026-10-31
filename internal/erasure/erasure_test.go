package erasure

import (
	"testing"

	"github.com/google/uuid"
)

// Sources run by name, whatever order they were registered in, and the core
// source last, so its scrub of the contacts comes after every match by
// address.
func TestRegistryOrder(t *testing.T) {
	r := New(Source{Name: "audience"}, Source{Name: "push"}, Source{Name: "earn"}, Source{Name: "channels"})
	var got []string
	for _, s := range r.Sources() {
		got = append(got, s.Name)
	}
	want := []string{"channels", "earn", "push", "audience"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order %v, want %v", got, want)
		}
	}
}

func TestRegistryRefusesDuplicates(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("two sources with one name were accepted")
		}
	}()
	New(Source{Name: "audience"}, Source{Name: "earn"}, Source{Name: "earn"})
}

func TestBatchMatches(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	batch := &Batch{Subjects: []Subject{{ID: a, Email: " Sam@Example.com ", Phone: "+13125550123"}, {ID: b}}}
	if ids := batch.IDs(); len(ids) != 2 || ids[0] != a || ids[1] != b {
		t.Fatalf("ids %v", ids)
	}
	if e := batch.Emails(); len(e) != 1 || e[0] != "sam@example.com" {
		t.Fatalf("emails %v", e)
	}
	if p := batch.Phones(); len(p) != 1 || p[0] != "+13125550123" {
		t.Fatalf("phones %v", p)
	}
}
