package workspaces_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

type sidebar struct {
	HiddenSections []string
}

func TestSidebarSections(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	// Every section shows by default, in the settings and with the workspace.
	var s, me sidebar
	c.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &s)
	c.Do("GET", ws+"/me", nil).MustStatus(t, 200).JSON(t, &me)
	if s.HiddenSections == nil || len(s.HiddenSections) != 0 || me.HiddenSections == nil || len(me.HiddenSections) != 0 {
		t.Fatalf("default: settings %v, me %v", s.HiddenSections, me.HiddenSections)
	}

	// Hiding keeps the sidebar's order, whatever the order of the changes.
	c.Do("PUT", ws+"/settings/sidebar/earn", map[string]any{"shown": false}).MustStatus(t, 200).JSON(t, &s)
	c.Do("PUT", ws+"/settings/sidebar/transactional", map[string]any{"shown": false}).MustStatus(t, 200).JSON(t, &s)
	if strings.Join(s.HiddenSections, ",") != "transactional,earn" {
		t.Fatalf("hidden: %v", s.HiddenSections)
	}
	c.Do("GET", ws+"/me", nil).MustStatus(t, 200).JSON(t, &me)
	if strings.Join(me.HiddenSections, ",") != "transactional,earn" {
		t.Fatalf("me: %v", me.HiddenSections)
	}
	// Setting the state a section already has changes nothing.
	c.Do("PUT", ws+"/settings/sidebar/earn", map[string]any{"shown": false}).MustStatus(t, 200).JSON(t, &s)
	c.Do("PUT", ws+"/settings/sidebar/transactional", map[string]any{"shown": true}).MustStatus(t, 200).JSON(t, &s)
	if strings.Join(s.HiddenSections, ",") != "earn" {
		t.Fatalf("shown again: %v", s.HiddenSections)
	}
	c.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &s)
	if strings.Join(s.HiddenSections, ",") != "earn" {
		t.Fatalf("settings: %v", s.HiddenSections)
	}

	// Each change is audited once, with what the sidebar hid before and after.
	rows, err := env.App.Pool.Query(context.Background(), `SELECT actor_name, before, after, metadata->>'summary', metadata->>'permission'
		FROM audit_events WHERE action = 'workspace.sidebar_updated' ORDER BY occurred_at, id`)
	if err != nil {
		t.Fatal(err)
	}
	type entry struct {
		Actor, Summary, Permission string
		Before, After              map[string][]string
	}
	var entries []entry
	for rows.Next() {
		var e entry
		var before, after []byte
		if err := rows.Scan(&e.Actor, &before, &after, &e.Summary, &e.Permission); err != nil {
			t.Fatal(err)
		}
		_ = json.Unmarshal(before, &e.Before)
		_ = json.Unmarshal(after, &e.After)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("audit entries: %+v", entries)
	}
	first, last := entries[0], entries[2]
	if first.Actor != "Linh Nguyen" || first.Summary != "Earn hidden from the sidebar" || first.Permission != "workspace:manage" ||
		len(first.Before["hidden_sections"]) != 0 || strings.Join(first.After["hidden_sections"], ",") != "earn" {
		t.Fatalf("first entry: %+v", first)
	}
	if last.Summary != "Transactional shown in the sidebar" || strings.Join(last.Before["hidden_sections"], ",") != "transactional,earn" ||
		strings.Join(last.After["hidden_sections"], ",") != "earn" {
		t.Fatalf("last entry: %+v", last)
	}

	// Only known sections; only people (and keys) with workspace:manage.
	c.Do("PUT", ws+"/settings/sidebar/campaigns", map[string]any{"shown": false}).MustStatus(t, 422)
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	if p := v.Do("PUT", ws+"/settings/sidebar/grow", map[string]any{"shown": false}).MustStatus(t, 403).Problem(t); !strings.Contains(p["detail"].(string), "workspace:manage") {
		t.Fatalf("viewer: %v", p)
	}
	// Members who can't change it still get what the sidebar hides.
	v.Do("GET", ws+"/me", nil).MustStatus(t, 200).JSON(t, &me)
	if strings.Join(me.HiddenSections, ",") != "earn" {
		t.Fatalf("viewer me: %v", me.HiddenSections)
	}
}
