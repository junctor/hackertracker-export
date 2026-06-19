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

func TestNormalizeFirestoreValueRejectsUnsupportedValues(t *testing.T) {
	_, err := normalizeFirestoreValue(map[string]any{"nested": struct{ Value string }{Value: "x"}})
	if err == nil {
		t.Fatal("normalizeFirestoreValue accepted unsupported struct")
	}
	if !strings.Contains(err.Error(), "nested: unsupported Firestore value") {
		t.Fatalf("error = %v", err)
	}
}
