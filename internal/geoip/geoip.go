// Package geoip finds the country and city of a reader's address in an
// operator-supplied MaxMind DB file (GeoLite2-City/Country, DB-IP Lite
// City/Country; config geoip.database). ReputeCast never downloads the
// file: the operator puts it in place, keeps it updated and restarts.
//
// Only the derived country code and English city name leave this package;
// the address itself is never stored for this (internal/tracking).
package geoip

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strings"
	"sync/atomic"

	"github.com/oschwald/maxminddb-golang/v2"
)

// DB is a GeoIP database held in memory. The nil DB and a DB that is still
// loading (or failed to) are off: Lookup finds nothing.
type DB struct {
	r atomic.Pointer[maxminddb.Reader]
}

// Open reads the file into memory and verifies it. Reading it whole, not
// memory-mapping it, means replacing the file on disk can't crash a
// running reader; verifying it means a corrupt or hostile file is refused
// before any lookup.
func Open(path string) (*DB, error) {
	r, err := open(path)
	if err != nil {
		return nil, err
	}
	d := &DB{}
	d.r.Store(r)
	return d, nil
}

func open(path string) (*maxminddb.Reader, error) {
	if path == "" {
		return nil, errors.New("geoip: no database file")
	}
	b, err := os.ReadFile(path) //nolint:gosec // G304: the operator's configured file
	if err != nil {
		return nil, fmt.Errorf("geoip: %w", err)
	}
	r, err := maxminddb.OpenBytes(b)
	if err != nil {
		return nil, fmt.Errorf("geoip: %s: %w", path, err)
	}
	if err := r.Verify(); err != nil {
		return nil, fmt.Errorf("geoip: %s: %w", path, err)
	}
	return r, nil
}

// Load returns at once and opens the file in the background, so a large or
// corrupt file neither delays start-up nor serves bad data: lookups find
// nothing until it is verified, and a file that can't be used is logged
// and stays off. An empty path is off (nil).
func Load(path string, log *slog.Logger) *DB {
	if path == "" {
		return nil
	}
	if log == nil {
		log = slog.Default()
	}
	d := &DB{}
	go func() {
		r, err := open(path)
		if err != nil {
			log.Error("geoip database unusable", "file", path, "err", err)
			return
		}
		d.r.Store(r)
		log.Info("geoip database loaded", "file", path, "type", r.Metadata.DatabaseType,
			"built", r.Metadata.BuildTime().UTC().Format("2006-01-02"))
	}()
	return d
}

// Loaded reports whether lookups can find places.
func (d *DB) Loaded() bool { return d != nil && d.r.Load() != nil }

// record is the part of a City or Country record that is read.
type record struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	City struct {
		Names struct {
			EN string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"city"`
}

// Lookup returns the ISO country code (upper case) and the English city
// name of an address; empty when the database is off or still loading,
// for private, loopback and unspecified addresses, and for addresses it
// doesn't know. A Country database finds no city.
func (d *DB) Lookup(ip netip.Addr) (country, city string) {
	if d == nil || !ip.IsValid() {
		return "", ""
	}
	r := d.r.Load()
	if r == nil {
		return "", ""
	}
	ip = ip.Unmap()
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsMulticast() {
		return "", ""
	}
	var rec record
	if err := r.Lookup(ip).Decode(&rec); err != nil {
		return "", ""
	}
	return strings.ToUpper(rec.Country.ISOCode), rec.City.Names.EN
}

// Describe is the database's type and build date ("GeoLite2-City, built
// 2026-11-04"), or "" when it is off or still loading.
func (d *DB) Describe() string {
	if d == nil {
		return ""
	}
	r := d.r.Load()
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%s, built %s", r.Metadata.DatabaseType, r.Metadata.BuildTime().UTC().Format("2006-01-02"))
}
