package export

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestStableJSONSortsAndSanitizes(t *testing.T) {
	value := map[string]any{
		"b": "one\t two   three \r\n",
		"a": map[string]any{
			"10": "ten",
			"2":  "two",
			"x":  "ex",
		},
	}
	got, err := StableJSON(value, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a":{"2":"two","10":"ten","x":"ex"},"b":"one  two three\n"}`
	if string(got) != want {
		t.Fatalf("StableJSON mismatch\nwant: %s\n got: %s", want, got)
	}
}

func TestStableJSONRejectsInvalidFloat(t *testing.T) {
	_, err := StableJSON(map[string]any{"value": math.NaN()}, false)
	if err == nil {
		t.Fatal("StableJSON accepted NaN")
	}
	if !strings.Contains(err.Error(), "unsupported floating-point value") {
		t.Fatalf("StableJSON error = %v", err)
	}
}

func TestStableJSONRejectsInvalidJSONNumber(t *testing.T) {
	_, err := StableJSON(map[string]any{"value": json.Number("NaN")}, false)
	if err == nil {
		t.Fatal("StableJSON accepted invalid JSON number")
	}
	if !strings.Contains(err.Error(), "unsupported JSON number") {
		t.Fatalf("StableJSON error = %v", err)
	}
}

func TestWriteArtifactsRequiresDerivedArtifacts(t *testing.T) {
	artifacts := Artifacts{
		Manifest: map[string]any{},
		Entities: map[string]any{
			"articles":      map[string]any{},
			"content":       map[string]any{},
			"documents":     map[string]any{},
			"events":        map[string]any{},
			"locations":     map[string]any{},
			"organizations": map[string]any{},
			"people":        map[string]any{},
			"tags":          map[string]any{},
		},
		Indexes: map[string]any{
			"eventsByDay": map[string]any{},
			"eventsByTag": map[string]any{},
		},
		Views: map[string]any{
			"announcementsList":  []any{},
			"bookmarkEventsById": map[string]any{},
			"contentCards":       []any{},
			"documentsList":      []any{},
			"locationCards":      []any{},
			"organizationsCards": map[string]any{},
			"peopleCards":        []any{},
			"searchData":         []any{},
			"scheduleDays":       []any{},
			"tagTypesBrowse":     []any{},
		},
		Derived: map[string]any{},
		Details: map[string]map[int]any{},
	}

	_, err := WriteArtifacts(t.TempDir(), artifacts)
	if err == nil {
		t.Fatal("WriteArtifacts accepted missing derived artifact")
	}
	if !strings.Contains(err.Error(), "missing generated artifact: tagIdsByLabel") {
		t.Fatalf("WriteArtifacts error = %v", err)
	}
}
