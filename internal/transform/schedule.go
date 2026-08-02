package transform

import (
	"cmp"
	"maps"
	"slices"
	"strings"
)

type builtIndexes struct {
	sessionsByDay map[string][]int
	sessionsByTag map[string][]int
}

type ScheduleBrowseSession struct {
	BeginDisplay          string       `json:"beginDisplay"`
	BeginIso              string       `json:"beginIso"`
	BeginTimestampSeconds int64        `json:"beginTimestampSeconds"`
	Color                 string       `json:"color"`
	ContentID             int          `json:"contentId"`
	EndDisplay            string       `json:"endDisplay"`
	EndIso                string       `json:"endIso"`
	EndTimestampSeconds   int64        `json:"endTimestampSeconds"`
	ID                    int          `json:"id"`
	LocationName          string       `json:"locationName"`
	PeopleText            *string      `json:"speakers"`
	TagCount              int          `json:"tagCount"`
	Tags                  []CompactTag `json:"tags"`
	Title                 string       `json:"title"`
}

type ScheduleDay struct {
	Day      string                  `json:"day"`
	Sessions []ScheduleBrowseSession `json:"sessions"`
}

type ScheduleSessionPosition struct {
	DayIndex     int `json:"dayIndex"`
	SessionIndex int `json:"sessionIndex"`
}

type ScheduleBrowse struct {
	Days                 []ScheduleDay                      `json:"days"`
	SessionPositionsByID map[string]ScheduleSessionPosition `json:"sessionPositionsById"`
}

type LocationCard struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	ParentID  *int    `json:"parentId"`
	ShortName *string `json:"shortName"`
}

func buildIndexes(st *stores, timezone string) builtIndexes {
	sessionsByDay := map[string][]int{}
	sessionsByTag := map[string][]int{}
	sessionStarts := map[int]int64{}

	for _, sessionID := range st.sessionIDs {
		session := st.sessionsByID[sessionID]
		sessionStarts[sessionID] = session.BeginTimestampSeconds
		if day := sessionDay(session.Begin, timezone); day != "" {
			sessionsByDay[day] = append(sessionsByDay[day], sessionID)
		}
		for _, tagID := range session.TagIDs {
			sessionsByTag[idKey(tagID)] = append(sessionsByTag[idKey(tagID)], sessionID)
		}
	}
	sortSessionIndex(sessionsByDay, sessionStarts)
	sortSessionIndex(sessionsByTag, sessionStarts)
	return builtIndexes{sessionsByDay: sessionsByDay, sessionsByTag: sessionsByTag}
}

func buildScheduleSessionViewModel(session SessionModel, st *stores) ScheduleBrowseSession {
	peopleNames := []string{}
	for _, personID := range session.PersonIDs {
		if person, ok := st.peopleByID[personID]; ok && person.Name != "" {
			peopleNames = append(peopleNames, person.Name)
		}
	}

	locationName := "Unknown location"
	if session.LocationID != nil {
		if location, ok := st.locationsByID[*session.LocationID]; ok && location.Name != "" {
			locationName = location.Name
		}
	}

	resolvedTags := tagsForIDs(session.TagIDs, st.tagsByID)
	slices.SortFunc(resolvedTags, compareTags)
	tags := make([]CompactTag, 0, len(resolvedTags))
	for _, tag := range resolvedTags {
		tags = append(tags, compactTag(tag))
	}
	tagCount := len(tags)
	if len(tags) > 4 {
		tags = tags[:4]
	}

	var peopleText *string
	if len(peopleNames) > 0 {
		text := strings.Join(peopleNames, ", ")
		peopleText = &text
	}

	return ScheduleBrowseSession{
		BeginDisplay:          firstNonEmpty(session.BeginDisplay, sessionTimeTable(session.Begin, true, "")),
		BeginIso:              firstNonEmpty(session.BeginIso, isoTime(session.Begin)),
		BeginTimestampSeconds: session.BeginTimestampSeconds,
		Color:                 session.Color,
		ContentID:             session.ContentID,
		EndDisplay:            firstNonEmpty(session.EndDisplay, sessionTimeTable(session.End, false, "")),
		EndIso:                firstNonEmpty(session.EndIso, isoTime(session.End)),
		EndTimestampSeconds:   session.EndTimestampSeconds,
		ID:                    session.ID,
		LocationName:          locationName,
		PeopleText:            peopleText,
		TagCount:              tagCount,
		Tags:                  tags,
		Title:                 session.Title,
	}
}

func buildPageReadyArtifacts(st *stores, indexes builtIndexes, timezone string) (map[string]any, map[string]map[int]any) {
	allSessions := make([]SessionModel, 0, len(st.sessionIDs))
	modelsBySessionID := map[int]ScheduleBrowseSession{}
	for _, sessionID := range st.sessionIDs {
		session := st.sessionsByID[sessionID]
		allSessions = append(allSessions, session)
		modelsBySessionID[sessionID] = buildScheduleSessionViewModel(session, st)
	}

	scheduleDays := buildAllScheduleDays(st, indexes, modelsBySessionID, timezone)
	scheduleBrowse := buildScheduleBrowse(scheduleDays)

	locationCards := []LocationCard{}
	for _, locationID := range st.locationIDs {
		location := st.locationsByID[locationID]
		locationCards = append(locationCards, LocationCard{
			ID:        location.ID,
			Name:      location.Name,
			ParentID:  location.ParentID,
			ShortName: location.ShortName,
		})
	}

	announcements := []ArticleModel{}
	for _, articleID := range st.articleIDs {
		announcements = append(announcements, st.articlesByID[articleID])
	}
	slices.SortFunc(announcements, func(a, b ArticleModel) int {
		return cmp.Or(
			cmp.Compare(valueOrZero(b.UpdatedAtMs), valueOrZero(a.UpdatedAtMs)),
			cmp.Compare(a.ID, b.ID),
		)
	})

	details := map[string]map[int]any{
		"content":       {},
		"people":        {},
		"documents":     {},
		"organizations": {},
	}
	for _, contentID := range st.contentIDs {
		details["content"][contentID] = buildContentDetail(st.contentByID[contentID], st, allSessions)
	}
	for _, personID := range st.peopleIDs {
		details["people"][personID] = buildPersonDetail(st.peopleByID[personID], st, allSessions)
	}
	for _, id := range st.documentIDs {
		details["documents"][id] = st.documentsByID[id]
	}
	for _, id := range st.organizationIDs {
		details["organizations"][id] = st.organizationsByID[id]
	}

	return map[string]any{
		"announcementsList":   announcements,
		"locationCards":       locationCards,
		"scheduleBrowse":      scheduleBrowse,
		"scheduleFilterIndex": FilterIndex{ItemCount: len(st.sessionIDs), ItemIDsByTag: indexes.sessionsByTag},
	}, details
}

func buildScheduleBrowse(scheduleDays []ScheduleDay) ScheduleBrowse {
	positions := map[string]ScheduleSessionPosition{}
	for dayIndex, day := range scheduleDays {
		for sessionIndex, session := range day.Sessions {
			positions[idKey(session.ID)] = ScheduleSessionPosition{DayIndex: dayIndex, SessionIndex: sessionIndex}
		}
	}
	return ScheduleBrowse{Days: scheduleDays, SessionPositionsByID: positions}
}

func buildAllScheduleDays(st *stores, indexes builtIndexes, modelsBySessionID map[int]ScheduleBrowseSession, timezone string) []ScheduleDay {
	keys := slices.Sorted(maps.Keys(indexes.sessionsByDay))
	days := []ScheduleDay{}
	for _, day := range keys {
		sessions := sessionsFromIDs(indexes.sessionsByDay[day], st.sessionsByID)
		models := modelsForSessions(sessions, modelsBySessionID)
		if len(models) > 0 {
			days = append(days, ScheduleDay{Day: day, Sessions: models})
		}
	}
	if len(keys) > 0 {
		return days
	}
	return buildScheduleDaysFromSessions(sessionsFromIDs(st.sessionIDs, st.sessionsByID), modelsBySessionID, timezone)
}

func buildScheduleDaysFromSessions(sessions []SessionModel, modelsBySessionID map[int]ScheduleBrowseSession, timezone string) []ScheduleDay {
	groups := map[string][]SessionModel{}
	for _, session := range sessions {
		day := sessionDay(session.Begin, timezone)
		if day == "" {
			continue
		}
		groups[day] = append(groups[day], session)
	}
	keys := slices.Sorted(maps.Keys(groups))
	out := []ScheduleDay{}
	for _, day := range keys {
		models := modelsForSessions(groups[day], modelsBySessionID)
		if len(models) > 0 {
			out = append(out, ScheduleDay{Day: day, Sessions: models})
		}
	}
	return out
}

func modelsForSessions(sessions []SessionModel, modelsBySessionID map[int]ScheduleBrowseSession) []ScheduleBrowseSession {
	sortSessions(sessions)
	models := []ScheduleBrowseSession{}
	for _, session := range sessions {
		if model, ok := modelsBySessionID[session.ID]; ok {
			models = append(models, model)
		}
	}
	return models
}

func sortSessions(sessions []SessionModel) {
	slices.SortFunc(sessions, func(a, b SessionModel) int {
		return cmp.Or(
			cmp.Compare(a.BeginTimestampSeconds, b.BeginTimestampSeconds),
			cmp.Compare(a.ID, b.ID),
		)
	})
}

func sessionsFromIDs(ids []int, byID map[int]SessionModel) []SessionModel {
	sessions := []SessionModel{}
	for _, id := range ids {
		if session, ok := byID[id]; ok {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
