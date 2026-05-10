package store

import "testing"

// TestIsSafeSQLIdentifier asserts the table-name guard rejects every payload
// shape that could escape the SELECT id FROM <name> position. This is the
// floor under ListIDs's identifier interpolation; if these fail the fallback
// parameterized path still runs, but the test pins the contract so future
// edits don't widen the regex by accident.
func TestIsSafeSQLIdentifier(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		// Valid identifiers — current spec resources and hand-added tables.
		{"resource_lower", "pricerunner", true},
		{"resource_underscored", "sep_products", true},
		{"resource_with_digit", "sep_offers_v2", true},
		{"underscore_prefix", "_internal", true},
		{"single_char", "a", true},

		// Invalid — metacharacters and SQL injection payloads.
		{"semicolon_drop", "pricerunner; DROP TABLE sep_products; --", false},
		{"quote", `bad"name`, false},
		{"backtick", "bad`name`", false},
		{"single_quote", "bad'name", false},
		{"space", "bad name", false},
		{"comment_marker", "bad--name", false},
		{"comma", "a,b", false},
		{"paren", "tbl()", false},
		{"empty", "", false},
		{"leading_digit", "1table", false},
		{"unicode_period", "tbl.name", false},
		{"newline", "tbl\nname", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSafeSQLIdentifier(tc.in); got != tc.want {
				t.Fatalf("isSafeSQLIdentifier(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
