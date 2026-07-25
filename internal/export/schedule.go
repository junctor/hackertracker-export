package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ScheduleExport struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Conference    ScheduleExportConference `json:"conference"`
	Sessions      []ScheduleExportSession  `json:"sessions"`
}

type ScheduleExportConference struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Timezone    string `json:"timezone"`
	URL         string `json:"url"`
}

type ScheduleExportSession struct {
	SessionID      string                         `json:"sessionId"`
	ContentID      string                         `json:"contentId"`
	Title          string                         `json:"title"`
	Description    string                         `json:"description,omitempty"`
	ContentType    string                         `json:"contentType,omitempty"`
	Start          string                         `json:"start"`
	End            string                         `json:"end"`
	Timezone       string                         `json:"timezone"`
	LocationID     string                         `json:"locationId,omitempty"`
	Location       string                         `json:"location,omitempty"`
	Speakers       []ScheduleExportSpeaker        `json:"speakers"`
	Organizations  []ScheduleExportOrganization   `json:"organizations"`
	Tags           []ScheduleExportTag            `json:"tags"`
	RelatedContent []ScheduleExportRelatedContent `json:"relatedContent"`
	LogoURL        string                         `json:"logoUrl,omitempty"`
	URL            string                         `json:"url"`
}

type ScheduleExportSpeaker struct {
	PersonID      string               `json:"personId"`
	Name          string               `json:"name"`
	Title         string               `json:"title,omitempty"`
	Bio           string               `json:"bio,omitempty"`
	Pronouns      string               `json:"pronouns,omitempty"`
	Organizations []string             `json:"organizations"`
	Links         []ScheduleExportLink `json:"links"`
}

type ScheduleExportOrganization struct {
	OrganizationID string               `json:"organizationId"`
	Name           string               `json:"name"`
	Description    string               `json:"description,omitempty"`
	URL            string               `json:"url,omitempty"`
	LogoURL        string               `json:"logoUrl,omitempty"`
	Links          []ScheduleExportLink `json:"links"`
}

type ScheduleExportTag struct {
	TagID string `json:"tagId"`
	Name  string `json:"name"`
}

type ScheduleExportRelatedContent struct {
	ContentID string `json:"contentId"`
	Title     string `json:"title"`
	URL       string `json:"url"`
}

type ScheduleExportLink struct {
	Label string `json:"label,omitempty"`
	URL   string `json:"url"`
}

var scheduleCSVHeader = []string{
	"conference_code",
	"conference_name",
	"conference_timezone",
	"session_id",
	"content_id",
	"title",
	"description",
	"content_type",
	"start",
	"end",
	"timezone",
	"location_id",
	"location",
	"speaker_ids",
	"speaker_names",
	"speaker_titles",
	"speaker_organizations",
	"speaker_bios",
	"organization_ids",
	"organization_names",
	"tag_ids",
	"tags",
	"related_content_ids",
	"related_content_titles",
	"related_content_urls",
	"logo_url",
	"url",
}

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
		if err := writer.Write(scheduleCSVRow(schedule.Conference, session)); err != nil {
			return fmt.Errorf("write CSV row %q: %w", path, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV %q: %w", path, err)
	}
	return nil
}

func scheduleCSVRow(conf ScheduleExportConference, session ScheduleExportSession) []string {
	return []string{
		conf.Code,
		conf.Name,
		conf.Timezone,
		session.SessionID,
		session.ContentID,
		session.Title,
		session.Description,
		session.ContentType,
		session.Start,
		session.End,
		session.Timezone,
		session.LocationID,
		session.Location,
		joinStrings(speakerValues(session.Speakers, func(speaker ScheduleExportSpeaker) string { return speaker.PersonID })),
		joinStrings(speakerValues(session.Speakers, func(speaker ScheduleExportSpeaker) string { return speaker.Name })),
		joinStrings(speakerValues(session.Speakers, func(speaker ScheduleExportSpeaker) string { return speaker.Title })),
		joinStrings(speakerValues(session.Speakers, func(speaker ScheduleExportSpeaker) string {
			return strings.Join(speaker.Organizations, ", ")
		})),
		joinStrings(speakerValues(session.Speakers, func(speaker ScheduleExportSpeaker) string { return speaker.Bio })),
		joinStrings(organizationValues(session.Organizations, func(org ScheduleExportOrganization) string { return org.OrganizationID })),
		joinStrings(organizationValues(session.Organizations, func(org ScheduleExportOrganization) string { return org.Name })),
		joinStrings(tagValues(session.Tags, func(tag ScheduleExportTag) string { return tag.TagID })),
		joinStrings(tagValues(session.Tags, func(tag ScheduleExportTag) string { return tag.Name })),
		joinStrings(relatedContentValues(session.RelatedContent, func(content ScheduleExportRelatedContent) string { return content.ContentID })),
		joinStrings(relatedContentValues(session.RelatedContent, func(content ScheduleExportRelatedContent) string { return content.Title })),
		joinStrings(relatedContentValues(session.RelatedContent, func(content ScheduleExportRelatedContent) string { return content.URL })),
		session.LogoURL,
		session.URL,
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

func relatedContentValues(contents []ScheduleExportRelatedContent, value func(ScheduleExportRelatedContent) string) []string {
	out := make([]string, 0, len(contents))
	for _, content := range contents {
		out = append(out, value(content))
	}
	return out
}

func joinStrings(values []string) string {
	return strings.Join(values, ";")
}
