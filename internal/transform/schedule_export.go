package transform

import (
	"cmp"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/junctor/hackertracker-export/internal/export"
	"github.com/junctor/hackertracker-export/pkg/hackertracker"
)

type scheduleExportSourceIndexes struct {
	orgIDsByName         map[string][]int
	orgIDsByOrganizerTag map[int][]int
	orgIDsByPersonID     map[int][]int
	speakerLinksByID     map[int][]LinkModel
	relatedContentByID   map[int][]int
}

func buildScheduleExport(conf hackertracker.Conference, data hackertracker.SourceData, st *stores) export.ScheduleExport {
	sourceIndexes := buildScheduleExportSourceIndexes(data, st)
	confCode := strings.TrimSpace(conf.Code)
	timezone := strings.TrimSpace(conf.Timezone)
	conference := export.ScheduleExportConference{
		Code:        confCode,
		Name:        cleanText(conf.Name),
		Description: cleanText(conf.Description),
		Timezone:    timezone,
		URL:         publicConferenceURL(confCode),
	}

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
		sessions = append(sessions, buildScheduleExportSession(confCode, timezone, session, content, st, sourceIndexes))
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
		SchemaVersion: 1,
		Conference:    conference,
		Sessions:      sessions,
	}
}

func buildScheduleExportSession(confCode, conferenceTimezone string, session SessionModel, content ContentModel, st *stores, sourceIndexes scheduleExportSourceIndexes) export.ScheduleExportSession {
	timezone := exportTimezone(session.TimezoneName, conferenceTimezone)
	locationID := ""
	locationName := ""
	if session.LocationID != nil {
		if location, ok := st.locationsByID[*session.LocationID]; ok {
			locationID = idKey(location.ID)
			locationName = cleanText(location.Name)
		}
	}

	speakers := scheduleExportSpeakers(session.PersonIDs, st, sourceIndexes)
	organizations := scheduleExportOrganizations(session, speakers, st, sourceIndexes)
	tags := scheduleExportTags(session.TagIDs, st)
	relatedIDs := content.RelatedContentIDs
	if ids, ok := sourceIndexes.relatedContentByID[content.ID]; ok {
		relatedIDs = ids
	}
	related := scheduleExportRelatedContent(confCode, relatedIDs, st)

	return export.ScheduleExportSession{
		SessionID:      idKey(session.ID),
		ContentID:      idKey(content.ID),
		Title:          cleanText(firstNonEmpty(session.Title, content.Title)),
		Description:    cleanText(content.Description),
		Start:          exportTimestamp(session.Begin, timezone),
		End:            exportTimestamp(session.End, timezone),
		Timezone:       timezone,
		LocationID:     locationID,
		Location:       locationName,
		Speakers:       speakers,
		Organizations:  organizations,
		Tags:           tags,
		RelatedContent: related,
		LogoURL:        normalizedAssetURL(content.LogoURL),
		URL:            publicContentURL(confCode, content.ID),
	}
}

func buildScheduleExportSourceIndexes(data hackertracker.SourceData, st *stores) scheduleExportSourceIndexes {
	indexes := scheduleExportSourceIndexes{
		orgIDsByName:         map[string][]int{},
		orgIDsByOrganizerTag: map[int][]int{},
		orgIDsByPersonID:     map[int][]int{},
		speakerLinksByID:     map[int][]LinkModel{},
		relatedContentByID:   map[int][]int{},
	}

	validContentIDs := map[int]bool{}
	for _, contentID := range st.contentIDs {
		validContentIDs[contentID] = true
	}
	for _, content := range data.Content {
		contentID, ok := normalizeID(content.ID)
		if !ok {
			continue
		}
		indexes.relatedContentByID[contentID] = uniqueIDs(content.RelatedContentIDs, validContentIDs)
	}

	for _, speaker := range data.Speakers {
		personID, ok := normalizeID(speaker.ID)
		if !ok {
			continue
		}
		links := linksToModels(speaker.Links)
		if strings.TrimSpace(speaker.Link) != "" {
			links = append(links, LinkModel{URL: speaker.Link})
		}
		indexes.speakerLinksByID[personID] = links
	}

	for _, org := range data.Organizations {
		orgID, ok := normalizeID(org.ID)
		if !ok {
			continue
		}
		if _, ok := st.organizationsByID[orgID]; !ok {
			continue
		}
		if nameKey := organizationNameKey(org.Name); nameKey != "" {
			indexes.orgIDsByName[nameKey] = append(indexes.orgIDsByName[nameKey], orgID)
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
	for _, ids := range indexes.orgIDsByName {
		sortOrgIDs(ids)
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
		links := person.Links
		if sourceLinks, ok := sourceIndexes.speakerLinksByID[personID]; ok {
			links = sourceLinks
		}
		speakers = append(speakers, export.ScheduleExportSpeaker{
			PersonID:      idKey(person.ID),
			Name:          cleanText(person.Name),
			Title:         cleanText(person.Title),
			Bio:           cleanText(person.Description),
			Pronouns:      cleanText(person.Pronouns),
			Organizations: cleanStringList(person.Affiliations),
			Links:         scheduleExportLinks(links),
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
		for _, name := range speaker.Organizations {
			orgIDs = append(orgIDs, sourceIndexes.orgIDsByName[organizationNameKey(name)]...)
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
		links := scheduleExportLinks(org.Links)
		orgs = append(orgs, export.ScheduleExportOrganization{
			OrganizationID: idKey(org.ID),
			Name:           cleanText(org.Name),
			Description:    cleanText(org.Description),
			URL:            firstLinkURL(links),
			LogoURL:        normalizedAssetURL(org.LogoURL),
			Links:          links,
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

func scheduleExportRelatedContent(confCode string, contentIDs []int, st *stores) []export.ScheduleExportRelatedContent {
	seen := map[int]bool{}
	related := []export.ScheduleExportRelatedContent{}
	for _, contentID := range contentIDs {
		if seen[contentID] {
			continue
		}
		content, ok := st.contentByID[contentID]
		if !ok {
			continue
		}
		seen[contentID] = true
		related = append(related, export.ScheduleExportRelatedContent{
			ContentID: idKey(content.ID),
			Title:     cleanText(content.Title),
			URL:       publicContentURL(confCode, content.ID),
		})
	}
	return related
}

func scheduleExportLinks(links []LinkModel) []export.ScheduleExportLink {
	seen := map[string]bool{}
	out := []export.ScheduleExportLink{}
	for _, link := range links {
		linkURL := cleanPublicURL(link.URL)
		if linkURL == "" || seen[linkURL] {
			continue
		}
		seen[linkURL] = true
		out = append(out, export.ScheduleExportLink{
			Label: cleanText(firstNonEmpty(link.Label, link.Type)),
			URL:   linkURL,
		})
	}
	return out
}

func firstLinkURL(links []export.ScheduleExportLink) string {
	for _, link := range links {
		if link.URL != "" {
			return link.URL
		}
	}
	return ""
}

func exportTimezone(sessionTimezone, conferenceTimezone string) string {
	for _, candidate := range []string{sessionTimezone, conferenceTimezone} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, err := time.LoadLocation(candidate); err == nil {
			return candidate
		}
	}
	return conferenceTimezone
}

func exportTimestamp(value, timezone string) string {
	t, ok := parseTime(value)
	if !ok {
		return ""
	}
	if loc, err := time.LoadLocation(timezone); err == nil {
		t = t.In(loc)
	}
	return t.Format(time.RFC3339)
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

func publicConferenceURL(confCode string) string {
	return fmt.Sprintf("https://info.defcon.org/%s/", strings.ToLower(strings.TrimSpace(confCode)))
}

func publicContentURL(confCode string, contentID int) string {
	u := url.URL{
		Scheme: "https",
		Host:   "info.defcon.org",
		Path:   "/" + strings.ToLower(strings.TrimSpace(confCode)) + "/content/",
	}
	query := url.Values{}
	query.Set("id", idKey(contentID))
	u.RawQuery = query.Encode()
	return u.String()
}

func cleanText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.TrimSpace(value)
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

func cleanPublicURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	switch parsed.Scheme {
	case "http", "https":
		return trimmed
	default:
		return ""
	}
}

func organizationNameKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
