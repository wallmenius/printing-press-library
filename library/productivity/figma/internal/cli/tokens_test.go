// Copyright 2026 giuliano-giacaglia. Licensed under Apache-2.0. See LICENSE.

package cli

import "testing"

func TestDiffVariables(t *testing.T) {
	a := []variable{
		{ID: "v1", Name: "color/brand/primary", ValuesByMode: map[string]any{"light": "#000"}},
		{ID: "v2", Name: "spacing/md", ValuesByMode: map[string]any{"default": float64(8)}},
		{ID: "v3", Name: "color/legacy", ValuesByMode: map[string]any{"light": "#fff"}},
	}
	b := []variable{
		{ID: "v1", Name: "color/brand/primary", ValuesByMode: map[string]any{"light": "#111"}},  // value changed
		{ID: "v2", Name: "spacing/medium", ValuesByMode: map[string]any{"default": float64(8)}}, // renamed
		{ID: "v4", Name: "spacing/lg", ValuesByMode: map[string]any{"default": float64(16)}},    // added
	}

	res := diffVariables(a, b)

	if len(res.Added) != 1 || res.Added[0].ID != "v4" {
		t.Errorf("Added: got %+v, want [v4]", res.Added)
	}
	if len(res.Removed) != 1 || res.Removed[0].ID != "v3" {
		t.Errorf("Removed: got %+v, want [v3]", res.Removed)
	}
	if len(res.Renamed) != 1 || res.Renamed[0].ID != "v2" || res.Renamed[0].NewName != "spacing/medium" {
		t.Errorf("Renamed: got %+v, want [v2 medium]", res.Renamed)
	}
	if len(res.ValueChanged) != 1 || res.ValueChanged[0].ID != "v1" {
		t.Errorf("ValueChanged: got %+v, want [v1]", res.ValueChanged)
	}
}

func TestDiffVariables_ModeAware(t *testing.T) {
	// Same ids, same names, but a new mode appears in b.
	a := []variable{{ID: "v1", Name: "color/x", ValuesByMode: map[string]any{"light": "#000"}}}
	b := []variable{{ID: "v1", Name: "color/x", ValuesByMode: map[string]any{"light": "#000", "dark": "#fff"}}}
	res := diffVariables(a, b)
	if len(res.ValueChanged) != 1 {
		t.Errorf("expected ValueChanged on new mode, got %+v", res)
	}
}

func TestDiffVariables_Stable(t *testing.T) {
	// Reversing input order must not affect output ordering.
	a := []variable{{ID: "z", Name: "z"}, {ID: "a", Name: "a"}}
	b := []variable{{ID: "a", Name: "a"}, {ID: "z", Name: "z"}}
	r1 := diffVariables(a, b)
	r2 := diffVariables(b, a)
	if len(r1.Added) != 0 || len(r1.Removed) != 0 || len(r2.Added) != 0 || len(r2.Removed) != 0 {
		t.Errorf("expected no diffs, got %+v / %+v", r1, r2)
	}
}
