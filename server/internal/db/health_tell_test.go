package db

import (
	"context"
	"testing"
)

func TestHealthTellStore(t *testing.T) {
	d, ctx := open(t), context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	first, created, err := d.InsertHealthTell(ctx, HealthTell{ID: "tell_b", ClientID: "client-first", Text: "first", Source: "untrusted"})
	must(err)
	if !created || first.Status != "received" || first.Source != "shortcut" || first.CreatedAt.IsZero() {
		t.Fatalf("first = %+v, created %v", first, created)
	}
	again, created, err := d.InsertHealthTell(ctx, HealthTell{ID: "tell_retry", ClientID: first.ClientID, Text: "different"})
	must(err)
	if created || again.ID != first.ID || again.Text != "first" {
		t.Fatalf("retry = %+v, created %v", again, created)
	}
	_, _, err = d.InsertHealthTell(ctx, HealthTell{ID: "tell_a", ClientID: "client-second", Text: "second"})
	must(err)
	_, err = d.pool.Exec(ctx, `UPDATE health_tells SET created_at = $1`, first.CreatedAt)
	must(err)
	rows, err := d.HealthTellsPending(ctx, 1)
	must(err)
	if len(rows) != 1 || rows[0].ID != "tell_a" {
		t.Fatalf("tie order / limit = %+v", rows)
	}
	_, err = d.pool.Exec(ctx, `UPDATE health_tells SET created_at = created_at - interval '1 minute' WHERE id = 'tell_b'`)
	must(err)
	rows, err = d.HealthTellsPending(ctx, 10)
	must(err)
	if len(rows) != 2 || rows[0].ID != "tell_b" {
		t.Fatalf("time order = %+v", rows)
	}
	for _, proposed := range []string{"reserved", "replacement"} {
		got, err := d.ReserveHealthTellSession(ctx, first.ID, proposed)
		must(err)
		if got != "reserved" {
			t.Fatalf("reservation = %q", got)
		}
	}
	changed, err := d.SetHealthTellResult(ctx, first.ID, "received", "", "", "refused", true)
	must(err)
	rows, err = d.HealthTellsPending(ctx, 10)
	must(err)
	if !changed || rows[0].Attempts != 1 || rows[0].Text != "first" || rows[0].Error != "refused" {
		t.Fatalf("attempt = %+v", rows)
	}
	changed, err = d.SetHealthTellResult(ctx, first.ID, "delivered", "computer", "reserved", "", false)
	must(err)
	if !changed {
		t.Fatal("delivery not recorded")
	}
	got, _, err := d.InsertHealthTell(ctx, HealthTell{ID: "retry-delivered", ClientID: first.ClientID})
	must(err)
	if got.Text != "" || got.Status != "delivered" || got.SessionID != "reserved" || got.MachineID != "computer" {
		t.Fatalf("delivered = %+v", got)
	}
	changed, err = d.SetHealthTellResult(ctx, first.ID, "failed", "", "", "late", true)
	must(err)
	if changed {
		t.Fatal("terminal row changed")
	}
	_, err = d.SetHealthTellResult(ctx, "tell_a", "failed", "", "", "expired", false)
	must(err)
	_, err = d.ReserveHealthTellSession(ctx, "tell_a", "too-late")
	must(err)
	got, _, err = d.InsertHealthTell(ctx, HealthTell{ID: "retry-failed", ClientID: "client-second"})
	must(err)
	if got.Text != "second" || got.NewSessionID != "" || got.Status != "failed" {
		t.Fatalf("failed = %+v", got)
	}
	rows, err = d.HealthTellsPending(ctx, 10)
	must(err)
	if len(rows) != 0 {
		t.Fatalf("terminal rows pending: %+v", rows)
	}
}
