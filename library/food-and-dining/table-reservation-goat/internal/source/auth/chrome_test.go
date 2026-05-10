package auth

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/browserutils/kooky"
)

// TestChromeCookieCandidatePathsIn verifies the filesystem walk finds both
// Chrome 96+ (<profile>/Network/Cookies) and pre-96 (<profile>/Cookies)
// layouts even when <root>/Local State doesn't list the profile in
// info_cache. This is the bug surfaced by a user whose macOS Chrome had
// `Default/Cookies` and `Profile 1/Cookies` on disk but kooky returned
// 0 Tock cookies — symptom of info_cache not listing those profiles.
func TestChromeCookieCandidatePathsIn(t *testing.T) {
	tmp := t.TempDir()

	// Layout: a Chrome-like root with both layouts present in different profiles,
	// plus a non-profile sibling directory that must be skipped.
	mustWriteFile := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	root := filepath.Join(tmp, "Chrome")
	// Default profile: only OLD layout (the failing case from the bug report)
	defaultCookies := filepath.Join(root, "Default", "Cookies")
	mustWriteFile(defaultCookies)
	// Profile 1: only NEW layout
	profile1Network := filepath.Join(root, "Profile 1", "Network", "Cookies")
	mustWriteFile(profile1Network)
	// Profile 2: BOTH layouts (e.g., a recently migrated profile)
	profile2Network := filepath.Join(root, "Profile 2", "Network", "Cookies")
	profile2Old := filepath.Join(root, "Profile 2", "Cookies")
	mustWriteFile(profile2Network)
	mustWriteFile(profile2Old)
	// Non-profile sibling: must NOT be enumerated as a profile.
	mustWriteFile(filepath.Join(root, "Crashpad", "Cookies"))
	mustWriteFile(filepath.Join(root, "GrShaderCache", "Cookies"))
	// A file (not a directory) at root level: must not crash the walk.
	mustWriteFile(filepath.Join(root, "Local State"))

	got := chromeCookieCandidatePathsIn([]string{root})

	want := []string{
		defaultCookies,
		profile1Network,
		profile2Network,
		profile2Old,
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("chromeCookieCandidatePathsIn:\n got:  %v\n want: %v", got, want)
	}

	// Excluded sibling must NOT appear.
	for _, g := range got {
		if strings.Contains(g, "Crashpad") || strings.Contains(g, "GrShaderCache") {
			t.Errorf("non-profile dir leaked into candidates: %q", g)
		}
	}
}

// TestChromeCookieCandidatePathsIn_MissingRoot verifies a non-existent root
// is silently skipped rather than failing the walk — Chrome Beta / Canary
// / Brave on machines that don't have them installed must be tolerated.
func TestChromeCookieCandidatePathsIn_MissingRoot(t *testing.T) {
	got := chromeCookieCandidatePathsIn([]string{"/nonexistent-path-aaa", "/nonexistent-path-bbb"})
	if len(got) != 0 {
		t.Errorf("expected zero candidates for non-existent roots; got %v", got)
	}
}

// TestDedupeCookies_KeysOnDomainNamePath verifies the dedupe collapses
// duplicate (Domain, Name, Path) entries and prefers the entry with the
// later Expires.
func TestDedupeCookies_KeysOnDomainNamePath(t *testing.T) {
	now := time.Now()
	earlier := now.Add(1 * time.Hour)
	later := now.Add(24 * time.Hour)

	a := []*kooky.Cookie{
		{Cookie: http.Cookie{Name: "session", Domain: ".opentable.com", Path: "/", Value: "old", Expires: earlier}},
		{Cookie: http.Cookie{Name: "csrf", Domain: ".opentable.com", Path: "/", Value: "x", Expires: later}},
	}
	b := []*kooky.Cookie{
		// Same key as a's "session" cookie but with a LATER expiry — should win.
		{Cookie: http.Cookie{Name: "session", Domain: ".opentable.com", Path: "/", Value: "new", Expires: later}},
		// Distinct (different Path)
		{Cookie: http.Cookie{Name: "session", Domain: ".opentable.com", Path: "/admin", Value: "admin-only", Expires: later}},
	}

	got := dedupeCookies(a, b)
	if len(got) != 3 {
		t.Fatalf("expected 3 deduped cookies; got %d (%+v)", len(got), got)
	}
	for _, c := range got {
		if c.Name == "session" && c.Path == "/" && c.Value != "new" {
			t.Errorf("dedupeCookies kept the older session cookie value=%q; expected the later-expiring 'new' value", c.Value)
		}
	}
}

// TestDedupeCookies_NilEntriesIgnored confirms nil entries (which kooky may
// occasionally emit) don't panic and don't pollute the output.
func TestDedupeCookies_NilEntriesIgnored(t *testing.T) {
	a := []*kooky.Cookie{
		nil,
		{Cookie: http.Cookie{Name: "x", Domain: "y", Path: "/"}},
		nil,
	}
	got := dedupeCookies(a)
	if len(got) != 1 {
		t.Errorf("expected nil entries to be skipped; got %d cookies", len(got))
	}
}
