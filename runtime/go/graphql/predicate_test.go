package graphql

import (
	"encoding/json"
	"testing"
)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPredicateSingleColumn(t *testing.T) {
	id := StringField{Col: "id"}
	if got := mustJSON(t, id.Eq("x")); got != `{"id":{"_eq":"x"}}` {
		t.Fatalf("Eq: %s", got)
	}
	if got := mustJSON(t, id.In("a", "b")); got != `{"id":{"_in":["a","b"]}}` {
		t.Fatalf("In: %s", got)
	}
	count := Int64Field{Col: "memberCount"}
	if got := mustJSON(t, count.Gt(1)); got != `{"memberCount":{"_gt":"1"}}` {
		t.Fatalf("Gt: %s", got) // Int64 marshals as a quoted string
	}
}

func TestPredicateCombinators(t *testing.T) {
	id := StringField{Col: "id"}
	name := StringField{Col: "name"}
	got := mustJSON(t, And(id.Eq("x"), name.Like("Bob%")))
	want := `{"_and":[{"id":{"_eq":"x"}},{"name":{"_like":"Bob%"}}]}`
	if got != want {
		t.Fatalf("And:\n got %s\nwant %s", got, want)
	}
	if got := mustJSON(t, Not(id.Eq("x"))); got != `{"_not":{"id":{"_eq":"x"}}}` {
		t.Fatalf("Not: %s", got)
	}
	if !IsOmitted(Predicate{}) {
		t.Fatal("zero Predicate should be omitted")
	}
	if IsOmitted(id.Eq("x")) {
		t.Fatal("a built Predicate should not be omitted")
	}
}

func TestRelation(t *testing.T) {
	email := StringField{Col: "email"}
	got := mustJSON(t, Relation("organisationMembers", email.Eq("a@b.com")))
	want := `{"organisationMembers":{"email":{"_eq":"a@b.com"}}}`
	if got != want {
		t.Fatalf("Relation:\n got %s\nwant %s", got, want)
	}
}

func TestOrderTerm(t *testing.T) {
	if got := mustJSON(t, StringField{Col: "displayName"}.Desc()); got != `{"displayName":"Desc"}` {
		t.Fatalf("Desc: %s", got)
	}
	if got := mustJSON(t, []OrderTerm{{"a", Asc}, {"b", Desc}}); got != `[{"a":"Asc"},{"b":"Desc"}]` {
		t.Fatalf("list: %s", got)
	}
}

func TestSetColumns(t *testing.T) {
	type patch struct {
		DisplayName string `json:"displayName,omitzero"`
		Slug        string `json:"slug,omitzero"`
		MemberCount Int64  `json:"memberCount,omitzero"`
	}
	got := mustJSON(t, SetColumns(patch{DisplayName: "BoB"}))
	if got != `{"displayName":{"set":"BoB"}}` {
		t.Fatalf("single set: %s", got)
	}
	// Unset fields are skipped; a set Int64 is included.
	got = mustJSON(t, SetColumns(patch{DisplayName: "BoB", MemberCount: 3}))
	if got != `{"displayName":{"set":"BoB"},"memberCount":{"set":"3"}}` {
		t.Fatalf("multi set: %s", got)
	}
	if got := mustJSON(t, SetColumns(patch{})); got != `{}` {
		t.Fatalf("empty patch: %s", got)
	}
}
