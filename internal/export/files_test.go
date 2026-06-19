package export

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWriteJSONPreservesSemanticContent(t *testing.T) {
	value := map[string]any{
		"b": "one\t two   three \r\n",
		"a": map[string]any{
			"10": "ten",
			"2":  "two",
			"x":  "ex",
		},
	}
	path := filepath.Join(t.TempDir(), "data.json")
	if err := WriteJSON(path, value); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	want := map[string]any{
		"b": "one\t two   three \r\n",
		"a": map[string]any{
			"10": "ten",
			"2":  "two",
			"x":  "ex",
		},
	}
	if !reflect.DeepEqual(decoded, want) {
		t.Fatalf("WriteJSON decoded value mismatch\nwant: %#v\n got: %#v", want, decoded)
	}
}

func TestWriteJSONRejectsInvalidFloat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	err := WriteJSON(path, map[string]any{"value": math.NaN()})
	if err == nil {
		t.Fatal("WriteJSON accepted NaN")
	}
}

func TestWriteJSONRejectsInvalidJSONNumber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	err := WriteJSON(path, map[string]any{"value": json.Number("NaN")})
	if err == nil {
		t.Fatal("WriteJSON accepted invalid JSON number")
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
			"tagTypes":      map[string]any{},
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
