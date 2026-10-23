package hostrepute_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/hostrepute/hrtest"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// A stored sync error reads in the reader's language; English readers read
// it as stored.
func TestSyncErrorIn(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	cases := map[string]string{
		"Couldn't reach api.hostrepute.com (timeout after 30 s, 3 attempts)": "Không kết nối được api.hostrepute.com (hết thời gian chờ sau 30 giây, 3 lần thử)",
		"Couldn't reach api.hostrepute.com (HTTP 503, 3 attempts)":           "Không kết nối được api.hostrepute.com (HTTP 503, 3 lần thử)",
		"Couldn't reach HostRepute":                                          "Không kết nối được HostRepute",
		"HostRepute sync failed: unexpected answer":                          "Đồng bộ HostRepute thất bại: unexpected answer",
		"HostRepute rejected the token. Reconnect to resume.":                "HostRepute đã từ chối token. Hãy kết nối lại để tiếp tục.",
		"HostRepute isn't connected.":                                        "HostRepute chưa được kết nối.",
		"something else":                                                     "something else",
	}
	for en, want := range cases {
		if got := hostrepute.SyncErrorIn(vi, en); got != want {
			t.Errorf("%q: %q, want %q", en, got, want)
		}
		if got := hostrepute.SyncErrorIn(i18n.Default, en); got != en {
			t.Errorf("English %q: %q", en, got)
		}
	}
	e := &hostrepute.UnavailableError{Host: "api.hostrepute.com", Reason: "timeout after 30 s", Attempts: 3}
	if e.In(i18n.Default) != e.Error() {
		t.Fatalf("English: %q, %q", e.In(i18n.Default), e.Error())
	}
}

// Settings › Integrations reads in the request's language: what each scope
// is for, a sync error stored while members read English, the problems, and
// the notification that asks to reconnect, kept for every member it reaches.
func TestHostReputeInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	fake := hrtest.New(t, "hr_live_sync")
	env.App.Cfg.HostRepute.APIURL = fake.APIURL()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	c.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 204)
	if pr := vi.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/api-key", map[string]any{"apiKey": "hr live"}).MustStatus(t, 422).Problem(t); pr["detail"] != "Hãy dán toàn bộ khóa, không có khoảng trắng. Khóa bắt đầu bằng hr_live_." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
	c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/api-key", map[string]any{"apiKey": "hr_live_sync"}).MustStatus(t, 200)
	ctx := context.Background()
	s := service(env)

	type view struct {
		HostRepute struct {
			SyncError string
			Scopes    []struct{ Key, Use string }
		}
	}
	var v view
	vi.Do("GET", "/workspaces/acme-newsletter/integrations", nil).MustStatus(t, 200).JSON(t, &v)
	uses := map[string]string{}
	for _, sc := range v.HostRepute.Scopes {
		uses[sc.Key] = sc.Use
	}
	if uses["monitor:read"] != "dữ liệu danh sách chặn, Postmaster, SNDS và DMARC" || uses["alerts:read"] != "sự cố" {
		t.Fatalf("scopes: %+v", v.HostRepute.Scopes)
	}

	fake.Down = true
	_ = s.Sync(ctx, hostrepute.SyncArgs{WorkspaceID: ws(env)}, false)
	var stored string
	_ = env.App.Pool.QueryRow(ctx, `SELECT last_sync_error FROM hostrepute_connections`).Scan(&stored)
	vi.Do("GET", "/workspaces/acme-newsletter/integrations", nil).MustStatus(t, 200).JSON(t, &v)
	if !strings.HasPrefix(stored, "Couldn't reach 127.0.0.1") || !strings.HasPrefix(v.HostRepute.SyncError, "Không kết nối được 127.0.0.1 (") {
		t.Fatalf("sync error: %q, read %q", stored, v.HostRepute.SyncError)
	}
	fake.Down = false

	fake.Revoked = true
	if err := s.Sync(ctx, hostrepute.SyncArgs{WorkspaceID: ws(env)}, true); err != nil {
		t.Fatal(err)
	}
	if pr := vi.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/assets", map[string]any{"asset": "mta.acme.io"}).MustStatus(t, 409).Problem(t); !strings.HasPrefix(pr["detail"].(string), "HostRepute đã từ chối token vào ") {
		t.Fatalf("Vietnamese reconnect problem: %v", pr)
	}
	var list struct {
		Items []struct{ Title, Body string }
	}
	vi.Do("GET", "/workspaces/acme-newsletter/notifications?limit=100", nil).MustStatus(t, 200).JSON(t, &list)
	found := false
	for _, n := range list.Items {
		if n.Title == "Cần kết nối lại HostRepute" && strings.HasPrefix(n.Body, "HostRepute đã từ chối token.") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no Vietnamese reconnect notification: %+v", list.Items)
	}
}
