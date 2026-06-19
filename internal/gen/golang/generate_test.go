package golang

import "testing"

// TestIdentifierKeywordSafe verifies that names reducing to Go keywords are made into
// valid package identifiers (a table named "Type" must not yield `package type`).
func TestIdentifierKeywordSafe(t *testing.T) {
	cases := map[string]string{
		"Type":           "type_",
		"Map":            "map_",
		"Func":           "func_",
		"Range":          "range_",
		"Select":         "select_",
		"Interface":      "interface_",
		"BufferSettings": "buffersettings",
		"123":            "res123",
		"":               "res",
	}
	for in, want := range cases {
		if got := identifier(in); got != want {
			t.Errorf("identifier(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestUniqueNameDedups verifies collision suffixing so two resources never share a
// package directory or aggregator field.
func TestUniqueNameDedups(t *testing.T) {
	used := map[string]bool{}
	if got := uniqueName("buffersettings", used); got != "buffersettings" {
		t.Fatalf("first = %q, want buffersettings", got)
	}
	if got := uniqueName("buffersettings", used); got != "buffersettings2" {
		t.Fatalf("second = %q, want buffersettings2", got)
	}
	if got := uniqueName("buffersettings", used); got != "buffersettings3" {
		t.Fatalf("third = %q, want buffersettings3", got)
	}
}
