package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/blobstore"
	"github.com/hostrepute/reputecast/internal/blobstore/blobstoretest"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/secrets"
)

func TestDoctor(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	cfg := *env.App.Cfg
	cfg.Files.Dir = t.TempDir()
	cfg.HTTP.PublicURL = "https://news.example.com"
	right := func(*config.Config) (*secrets.Box, error) { return env.App.Secrets, nil }

	var out bytes.Buffer
	if err := printChecks(&out, runDoctor(ctx, &cfg, right)); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	for _, want := range []string{"[ok  ] Database", "[ok  ] Schema", "[ok  ] Secret key", "opens the", "[ok  ] File storage", "[ok  ] GeoIP", "[ok  ] Public URL"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in\n%s", want, out.String())
		}
	}

	// A GeoIP file that isn't one: warned, the other checks still pass.
	geo := cfg
	geo.GeoIP.Database = filepath.Join(t.TempDir(), "GeoLite2-City.mmdb")
	if err := os.WriteFile(geo.GeoIP.Database, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := printChecks(&out, runDoctor(ctx, &geo, right)); err != nil || !strings.Contains(out.String(), "[warn] GeoIP") {
		t.Fatalf("bad GeoIP file: %v\n%s", err, out.String())
	}

	// A locked public URL only this machine reaches.
	locked := cfg
	locked.HTTP.PublicURL, locked.HTTP.PublicURLLocked = "http://localhost:8080", true
	out.Reset()
	_ = printChecks(&out, runDoctor(ctx, &locked, right))
	if !strings.Contains(out.String(), "[warn] Public URL") || !strings.Contains(out.String(), "http.public_url_locked") {
		t.Fatalf("locked localhost:\n%s", out.String())
	}

	// Restored with another key: the sealed values don't open.
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	other, _ := secrets.New(k)
	wrong := func(*config.Config) (*secrets.Box, error) { return other, nil }
	out.Reset()
	err := printChecks(&out, runDoctor(ctx, &cfg, wrong))
	if err == nil || !strings.Contains(out.String(), "[FAIL] Secret key") || !strings.Contains(out.String(), "is this the key backed up with the database?") {
		t.Fatalf("wrong key: %v\n%s", err, out.String())
	}

	// A key change not resealed yet: values on the previous key.
	old, _ := config.DecodeKey(env.App.Cfg.Secrets.Key)
	rotated, _ := secrets.New(k, old)
	out.Reset()
	_ = printChecks(&out, runDoctor(ctx, &cfg, func(*config.Config) (*secrets.Box, error) { return rotated, nil }))
	if !strings.Contains(out.String(), "[warn] Previous keys") || !strings.Contains(out.String(), "secrets reseal") {
		t.Fatalf("previous key:\n%s", out.String())
	}

	// The database can't be reached: the file storage is still checked.
	cfg.Database.URL = "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"
	out.Reset()
	if err := printChecks(&out, runDoctor(ctx, &cfg, right)); err == nil || !strings.Contains(out.String(), "[FAIL] Database") || !strings.Contains(out.String(), "File storage") {
		t.Fatalf("no database: %v\n%s", err, out.String())
	}
}

// The file storage: a files folder that doesn't exist yet warns, one that
// does is counted; a bucket must answer and take a test object, which
// doesn't stay.
func TestDoctorFileStorage(t *testing.T) {
	ctx := context.Background()
	cfg := config.Defaults()
	cfg.Files.Dir = filepath.Join(t.TempDir(), "files")
	if c := filesCheck(ctx, &cfg); c.level != checkWarn || !strings.Contains(c.text, "doesn't exist yet") {
		t.Fatalf("no folder: %+v", c)
	}
	if err := (blobstore.Local{Dir: cfg.Files.Dir}).Put(ctx, "ws/ab/one", strings.NewReader("12345"), 5, ""); err != nil {
		t.Fatal(err)
	}
	if c := filesCheck(ctx, &cfg); c.level != checkOK || !strings.Contains(c.text, "1 files, 5 B") {
		t.Fatalf("a folder: %+v", c)
	}
	s3, ok := blobstoretest.S3Config(t)
	if !ok {
		t.Skip(blobstoretest.EnvEndpoint + " isn't set: the bucket checks need an S3-compatible server")
	}
	cfg.Storage = config.StorageConfig{Backend: config.StorageS3, S3: s3}
	c := filesCheck(ctx, &cfg)
	if c.level != checkOK || !strings.Contains(c.text, "s3 bucket "+s3.Bucket) || !strings.Contains(c.text, "0 objects") {
		t.Fatalf("a bucket: %+v", c)
	}
	missing := cfg
	missing.Storage.S3.Bucket = "reputecast-no-such-bucket"
	if c := filesCheck(ctx, &missing); c.level != checkFail || !strings.Contains(c.text, "doesn't exist") {
		t.Fatalf("a missing bucket: %+v", c)
	}
}

func TestHealthcheck(t *testing.T) {
	if got := localURLs(":8080", "0.0.0.0:9464", "/readyz"); len(got) != 2 || got[0] != "http://127.0.0.1:8080/readyz" || got[1] != "http://127.0.0.1:9464/readyz" {
		t.Fatalf("urls: %v", got)
	}
	status := http.StatusServiceUnavailable
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte("database schema behind this release"))
	}))
	defer srv.Close()
	run := func(args ...string) (string, error) {
		cmd := newHealthcheckCmd(&rootOpts{})
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		err := cmd.Execute()
		return out.String(), err
	}
	if _, err := run("--url", srv.URL+"/readyz"); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("not ready: %v", err)
	}
	status = http.StatusOK
	if out, err := run("--url", srv.URL+"/readyz"); err != nil || !strings.Contains(out, "schema") {
		t.Fatalf("ready: %v %q", err, out)
	}
	if _, err := run("--url", "http://127.0.0.1:1/readyz"); err == nil || !strings.Contains(err.Error(), "no ReputeCast process answered") {
		t.Fatalf("nothing listening: %v", err)
	}
}
