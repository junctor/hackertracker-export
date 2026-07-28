package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ScheduleExport struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Metadata      ScheduleExportMetadata  `json:"metadata"`
	Sessions      []ScheduleExportSession `json:"sessions"`
}

type ScheduleExportMetadata struct {
	ConferenceCode           string `json:"conferenceCode,omitempty"`
	ConferenceName           string `json:"conferenceName,omitempty"`
	ConferenceTimezone       string `json:"conferenceTimezone,omitempty"`
	DescriptionSnippet       string `json:"descriptionSnippet,omitempty"`
	SessionCount             int    `json:"sessionCount"`
	TimeZoneForTimestamps    string `json:"timeZoneForTimestamps"`
	TimeFormat               string `json:"timeFormat"`
	TextSnippetMaxCharacters int    `json:"textSnippetMaxCharacters"`
}

type ScheduleExportSession struct {
	SessionID          string                       `json:"sessionId"`
	ContentID          string                       `json:"contentId"`
	Title              string                       `json:"title"`
	DescriptionSnippet string                       `json:"descriptionSnippet,omitempty"`
	Start              string                       `json:"start"`
	End                string                       `json:"end"`
	Location           string                       `json:"location,omitempty"`
	Speakers           []ScheduleExportSpeaker      `json:"speakers"`
	Organizations      []ScheduleExportOrganization `json:"organizations"`
	Tags               []ScheduleExportTag          `json:"tags"`
}

type ScheduleExportSpeaker struct {
	PersonID   string `json:"-"`
	Name       string `json:"name"`
	BioSnippet string `json:"bioSnippet,omitempty"`
}

type ScheduleExportOrganization struct {
	OrganizationID     string `json:"-"`
	Name               string `json:"name"`
	DescriptionSnippet string `json:"descriptionSnippet,omitempty"`
}

type ScheduleExportTag struct {
	TagID string `json:"-"`
	Name  string `json:"name"`
}

var scheduleCSVHeader = []string{
	"session_id",
	"content_id",
	"start_utc",
	"end_utc",
	"location_name",
	"title",
	"speaker_names",
	"organization_names",
	"tag_names",
	"description_snippet",
}

const ScheduleTextSnippetLength = 1200

func writeScheduleCSV(path string, schedule ScheduleExport) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", dir, err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close CSV %q: %w", path, closeErr)
		}
	}()

	writer := csv.NewWriter(file)
	writer.UseCRLF = false
	if err := writer.Write(scheduleCSVHeader); err != nil {
		return fmt.Errorf("write CSV header %q: %w", path, err)
	}
	for _, session := range schedule.Sessions {
		if err := writer.Write(scheduleCSVRow(session)); err != nil {
			return fmt.Errorf("write CSV row %q: %w", path, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV %q: %w", path, err)
	}
	return nil
}

func scheduleCSVRow(session ScheduleExportSession) []string {
	return []string{
		csvCell(session.SessionID),
		csvCell(session.ContentID),
		csvCell(session.Start),
		csvCell(session.End),
		csvCell(session.Location),
		csvCell(session.Title),
		joinStrings(speakerValues(session.Speakers, func(speaker ScheduleExportSpeaker) string { return speaker.Name })),
		joinStrings(organizationValues(session.Organizations, func(org ScheduleExportOrganization) string { return org.Name })),
		joinStrings(tagValues(session.Tags, func(tag ScheduleExportTag) string { return tag.Name })),
		TextSnippet(session.DescriptionSnippet, ScheduleTextSnippetLength),
	}
}

func speakerValues(speakers []ScheduleExportSpeaker, value func(ScheduleExportSpeaker) string) []string {
	out := make([]string, 0, len(speakers))
	for _, speaker := range speakers {
		out = append(out, value(speaker))
	}
	return out
}

func organizationValues(orgs []ScheduleExportOrganization, value func(ScheduleExportOrganization) string) []string {
	out := make([]string, 0, len(orgs))
	for _, org := range orgs {
		out = append(out, value(org))
	}
	return out
}

func tagValues(tags []ScheduleExportTag, value func(ScheduleExportTag) string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, value(tag))
	}
	return out
}

func joinStrings(values []string) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, csvCell(value))
	}
	return strings.Join(out, ";")
}

func TextSnippet(value string, maxRunes int) string {
	value = csvCell(value)
	if maxRunes <= 0 || len([]rune(value)) <= maxRunes {
		return value
	}

	if maxRunes <= len("...") {
		return string([]rune(value)[:maxRunes])
	}

	limit := maxRunes - len("...")
	runes := []rune(value)
	cut := limit
	for i := limit; i > 0; i-- {
		if runes[i-1] == ' ' {
			cut = i - 1
			break
		}
	}
	if cut == 0 {
		cut = limit
	}
	return strings.TrimSpace(string(runes[:cut])) + "..."
}

func csvCell(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
