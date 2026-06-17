package transform

import (
	"testing"
	"time"

	"github.com/junctor/hackertracker-export/pkg/hackertracker"
)

func TestBuildInfoArtifacts(t *testing.T) {
	conf := hackertracker.Conference{Code: "TEST", Name: "Test Con", Timezone: "America/Los_Angeles"}
	data := hackertracker.SourceData{
		Content: []hackertracker.Content{{
			ID:          100,
			Title:       "Opening",
			Description: "Welcome",
			People:      []hackertracker.ContentPerson{{PersonID: 200, SortOrder: 1}},
			Sessions:    []hackertracker.Session{{SessionID: 300, BeginTSZ: "2026-08-06T17:00:00Z", EndTSZ: "2026-08-06T18:00:00Z", LocationID: 400}},
			TagIDs:      []any{500},
		}},
		Locations: []hackertracker.Location{{ID: 400, Name: "Room 1"}},
		Speakers:  []hackertracker.Speaker{{ID: 200, Name: "Alice", ContentIDs: []any{100}}},
		TagTypes: []hackertracker.TagType{{
			ID: 600, Label: "Track", Category: "content", SortOrder: 1, IsBrowsable: true,
			Tags: []hackertracker.Tag{{ID: 500, Label: "Talk", ColorBackground: "#000000", ColorForeground: "#ffffff", SortOrder: 1}},
		}},
		Organizations: []hackertracker.Organization{{ID: 700, Name: "Org", TagIDs: []any{500}}},
		Documents:     []hackertracker.Document{{ID: 800, TitleText: "Guide", UpdatedTSZ: "2026-08-01T00:00:00Z"}},
		Articles:      []hackertracker.Article{{ID: 900, Name: "News", UpdatedTSZ: "2026-08-02T00:00:00Z"}},
	}
	artifacts, err := Build(conf, data, BuildOptions{BuildTimestamp: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}

	events := artifacts.Entities["events"].(map[string]any)
	eventIDs := events["allIds"].([]int)
	if len(eventIDs) != 1 || eventIDs[0] != 300 {
		t.Fatalf("event ids = %#v", eventIDs)
	}
	days := artifacts.Indexes["eventsByDay"].(map[string][]int)
	if got := days["2026-08-06"]; len(got) != 1 || got[0] != 300 {
		t.Fatalf("eventsByDay = %#v", days)
	}
	views := artifacts.Views
	if len(views["scheduleDays"].([]any)) != 1 {
		t.Fatalf("scheduleDays = %#v", views["scheduleDays"])
	}
	if artifacts.Details["content"][100] == nil {
		t.Fatal("missing content detail")
	}
	manifest := artifacts.Manifest.(map[string]any)
	if manifest["buildTimestamp"] != "2026-01-02T03:04:05.000Z" {
		t.Fatalf("buildTimestamp = %v", manifest["buildTimestamp"])
	}
}
