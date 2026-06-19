package hackertracker

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeSourceDataPreservesExactNumbers(t *testing.T) {
	raw := map[string][]map[string]any{
		"content": {{
			"id":      int64(9007199254740993),
			"title":   "Large ID",
			"tag_ids": []any{int64(9007199254740995)},
			"links": []any{map[string]any{
				"label": "Site",
				"type":  "web",
				"url":   "https://example.test",
			}},
		}},
	}

	data, err := DecodeSourceData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := data.Content[0].ID; got != json.Number("9007199254740993") {
		t.Fatalf("content ID = %q", got)
	}
	if got := data.Content[0].TagIDs[0]; got != json.Number("9007199254740995") {
		t.Fatalf("tag ID = %q", got)
	}
	if got := data.Content[0].Links[0].URL; got != "https://example.test" {
		t.Fatalf("link URL = %q", got)
	}
}

func TestDecodeSourceDataDecodesSpeakerAffiliations(t *testing.T) {
	raw := map[string][]map[string]any{
		"speakers": {{
			"id":   int64(1),
			"name": "Alice",
			"affiliations": []any{
				"Legacy Org",
				map[string]any{"organization": "DDAS", "title": "Vice-President"},
				map[string]any{"organization": "", "title": "Independent Researcher"},
			},
		}},
	}

	data, err := DecodeSourceData(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := []string(data.Speakers[0].Affiliations)
	want := []string{"Legacy Org", "DDAS", "Independent Researcher"}
	if len(got) != len(want) {
		t.Fatalf("affiliations = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("affiliations = %#v, want %#v", got, want)
		}
	}
}

func TestNormalizeFirestoreValueRejectsUnsupportedValues(t *testing.T) {
	_, err := normalizeFirestoreValue(map[string]any{"nested": struct{ Value string }{Value: "x"}})
	if err == nil {
		t.Fatal("normalizeFirestoreValue accepted unsupported struct")
	}
	if !strings.Contains(err.Error(), "nested: unsupported Firestore value") {
		t.Fatalf("error = %v", err)
	}
}
