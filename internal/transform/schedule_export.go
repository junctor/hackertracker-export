package transform

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/junctor/hackertracker-export/internal/export"
	"github.com/junctor/hackertracker-export/pkg/hackertracker"
)

type scheduleExportSourceIndexes struct {
	orgIDsByOrganizerTag map[int][]int
	orgIDsByPersonID     map[int][]int
}

func buildScheduleExport(conf hackertracker.Conference, data hackertracker.SourceData, st *stores) export.ScheduleExport {
	sourceIndexes := buildScheduleExportSourceIndexes(data, st)

	sessions := make([]export.ScheduleExportSession, 0, len(st.sessionIDs))
	for _, sessionID := range st.sessionIDs {
		session, ok := st.sessionsByID[sessionID]
		if !ok {
			continue
		}
		content, ok := st.contentByID[session.ContentID]
		if !ok {
			continue
		}
		sessions = append(sessions, buildScheduleExportSession(session, content, st, sourceIndexes))
	}
	slices.SortFunc(sessions, func(a, b export.ScheduleExportSession) int {
		return cmp.Or(
			compareExportTime(a.Start, b.Start),
			compareExportTime(a.End, b.End),
			alphaCompare(a.Title, b.Title),
			compareExportID(a.SessionID, b.SessionID),
		)
	})

	return export.ScheduleExport{
		SchemaVersion: 4,
		Metadata: export.ScheduleExportMetadata{
			ConferenceCode:           cleanText(conf.Code),
			ConferenceName:           cleanText(conf.Name),
			ConferenceTimezone:       cleanText(conf.Timezone),
			DescriptionSnippet:       export.TextSnippet(conf.Description, export.ScheduleTextSnippetLength),
			SessionCount:             len(sessions),
			TimeZoneForTimestamps:    "UTC",
			TimeFormat:               "RFC3339",
			TextSnippetMaxCharacters: export.ScheduleTextSnippetLength,
		},
		Sessions: sessions,
	}
}

func buildScheduleExportSession(session SessionModel, content ContentModel, st *stores, sourceIndexes scheduleExportSourceIndexes) export.ScheduleExportSession {
	locationName := ""
	if session.LocationID != nil {
		if location, ok := st.locationsByID[*session.LocationID]; ok {
			locationName = cleanText(location.Name)
		}
	}

	speakers := scheduleExportSpeakers(session.PersonIDs, st, sourceIndexes)
	organizations := scheduleExportOrganizations(session, speakers, st, sourceIndexes)
	tags := scheduleExportTags(session.TagIDs, st)

	return export.ScheduleExportSession{
		SessionID:          idKey(session.ID),
		ContentID:          idKey(content.ID),
		Title:              cleanText(firstNonEmpty(session.Title, content.Title)),
		DescriptionSnippet: export.TextSnippet(content.Description, export.ScheduleTextSnippetLength),
		Start:              exportTimestamp(session.Begin),
		End:                exportTimestamp(session.End),
		Location:           locationName,
		Speakers:           speakers,
		Organizations:      organizations,
		Tags:               tags,
	}
}

func buildScheduleExportSourceIndexes(data hackertracker.SourceData, st *stores) scheduleExportSourceIndexes {
	indexes := scheduleExportSourceIndexes{
		orgIDsByOrganizerTag: map[int][]int{},
		orgIDsByPersonID:     map[int][]int{},
	}

	for _, org := range data.Organizations {
		orgID, ok := normalizeID(org.ID)
		if !ok {
			continue
		}
		if _, ok := st.organizationsByID[orgID]; !ok {
			continue
		}
		if org.TagIDAsOrganizer != nil {
			if tagID, ok := normalizeID(*org.TagIDAsOrganizer); ok {
				indexes.orgIDsByOrganizerTag[tagID] = append(indexes.orgIDsByOrganizerTag[tagID], orgID)
			}
		}
		for _, personID := range sourceOrganizationPersonIDs(org.People) {
			indexes.orgIDsByPersonID[personID] = append(indexes.orgIDsByPersonID[personID], orgID)
		}
	}
	sortScheduleExportSourceIndexes(indexes, st)
	return indexes
}

func sortScheduleExportSourceIndexes(indexes scheduleExportSourceIndexes, st *stores) {
	sortOrgIDs := func(ids []int) {
		slices.SortFunc(ids, func(a, b int) int {
			left := st.organizationsByID[a]
			right := st.organizationsByID[b]
			return cmp.Or(
				alphaCompare(left.Name, right.Name),
				cmp.Compare(left.ID, right.ID),
			)
		})
	}
	for _, ids := range indexes.orgIDsByOrganizerTag {
		sortOrgIDs(ids)
	}
	for _, ids := range indexes.orgIDsByPersonID {
		sortOrgIDs(ids)
	}
}

func sourceOrganizationPersonIDs(people []map[string]any) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, person := range people {
		for _, key := range []string{"person_id", "personId", "speaker_id", "speakerId", "id"} {
			id, ok := normalizeID(person[key])
			if !ok || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, id)
			break
		}
	}
	return out
}

func scheduleExportSpeakers(personIDs []int, st *stores, sourceIndexes scheduleExportSourceIndexes) []export.ScheduleExportSpeaker {
	speakers := []export.ScheduleExportSpeaker{}
	seen := map[int]bool{}
	for _, personID := range personIDs {
		if seen[personID] {
			continue
		}
		person, ok := st.peopleByID[personID]
		if !ok {
			continue
		}
		seen[personID] = true
		speakers = append(speakers, export.ScheduleExportSpeaker{
			PersonID:   idKey(person.ID),
			Name:       cleanText(person.Name),
			BioSnippet: export.TextSnippet(person.Description, export.ScheduleTextSnippetLength),
		})
	}
	return speakers
}

func scheduleExportOrganizations(session SessionModel, speakers []export.ScheduleExportSpeaker, st *stores, sourceIndexes scheduleExportSourceIndexes) []export.ScheduleExportOrganization {
	orgIDs := []int{}
	for _, tagID := range session.TagIDs {
		orgIDs = append(orgIDs, sourceIndexes.orgIDsByOrganizerTag[tagID]...)
	}
	for _, speaker := range speakers {
		personID, err := strconv.Atoi(speaker.PersonID)
		if err == nil {
			orgIDs = append(orgIDs, sourceIndexes.orgIDsByPersonID[personID]...)
		}
	}

	seen := map[int]bool{}
	orgs := []export.ScheduleExportOrganization{}
	for _, orgID := range orgIDs {
		if seen[orgID] {
			continue
		}
		org, ok := st.organizationsByID[orgID]
		if !ok {
			continue
		}
		seen[orgID] = true
		orgs = append(orgs, export.ScheduleExportOrganization{
			OrganizationID:     idKey(org.ID),
			Name:               cleanText(org.Name),
			DescriptionSnippet: export.TextSnippet(org.Description, export.ScheduleTextSnippetLength),
		})
	}
	slices.SortFunc(orgs, func(a, b export.ScheduleExportOrganization) int {
		return cmp.Or(
			alphaCompare(a.Name, b.Name),
			strings.Compare(a.OrganizationID, b.OrganizationID),
		)
	})
	return orgs
}

func scheduleExportTags(tagIDs []int, st *stores) []export.ScheduleExportTag {
	tags := tagsForIDs(tagIDs, st.tagsByID)
	slices.SortFunc(tags, compareTags)
	out := make([]export.ScheduleExportTag, 0, len(tags))
	for _, tag := range tags {
		if strings.TrimSpace(tag.Label) == "" {
			continue
		}
		out = append(out, export.ScheduleExportTag{
			TagID: idKey(tag.ID),
			Name:  cleanText(tag.Label),
		})
	}
	return out
}

func exportTimestamp(value string) string {
	t, ok := parseTime(value)
	if !ok {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func compareExportTime(a, b string) int {
	left, leftOK := parseTime(a)
	right, rightOK := parseTime(b)
	if leftOK && rightOK {
		return left.Compare(right)
	}
	if leftOK != rightOK {
		if leftOK {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

func compareExportID(a, b string) int {
	left, leftErr := strconv.Atoi(a)
	right, rightErr := strconv.Atoi(b)
	if leftErr == nil && rightErr == nil {
		return cmp.Compare(left, right)
	}
	return strings.Compare(a, b)
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func cleanStringList(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = cleanText(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
