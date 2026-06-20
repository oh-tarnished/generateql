package param

import (
	"encoding/json"
	"testing"
)

// filter mirrors a generated input type: optional scalars as Opt, a nested input value,
// and a slice, all with the omitzero tag.
type filter struct {
	Eq   Opt[string] `json:"_eq,omitzero"`
	N    Opt[int]    `json:"_n,omitzero"`
	Not  inner       `json:"_not,omitzero"`
	List []string    `json:"_in,omitzero"`
}

type inner struct {
	Eq Opt[string] `json:"_eq,omitzero"`
}

func TestMarshalOmitsUnset(t *testing.T) {
	got, err := json.Marshal(filter{})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{}" {
		t.Fatalf("empty filter should marshal to {}, got %s", got)
	}
}

func TestMarshalKeepsSetZeroValue(t *testing.T) {
	got, err := json.Marshal(filter{Eq: NewOpt(""), N: NewOpt(0)})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"_eq":"","_n":0}` {
		t.Fatalf("set zero values must encode explicitly, got %s", got)
	}
}

func TestMarshalNestedAndSlice(t *testing.T) {
	got, err := json.Marshal(filter{
		Not:  inner{Eq: NewOpt("x")},
		List: []string{"a", "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"_not":{"_eq":"x"},"_in":["a","b"]}`
	if string(got) != want {
		t.Fatalf("nested/slice mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	in := filter{Eq: NewOpt("hello"), N: NewOpt(42)}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out filter
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if v, ok := out.Eq.Get(); !ok || v != "hello" {
		t.Fatalf("Eq round-trip failed: %v %v", v, ok)
	}
	if out.N.Or(-1) != 42 {
		t.Fatalf("N round-trip failed: %d", out.N.Or(-1))
	}
}

func TestIsOmitted(t *testing.T) {
	if !IsOmitted(inner{}) {
		t.Fatal("zero struct should be omitted")
	}
	if IsOmitted(inner{Eq: NewOpt("x")}) {
		t.Fatal("set struct should not be omitted")
	}
	if !IsOmitted([]string(nil)) {
		t.Fatal("nil slice should be omitted")
	}
}
