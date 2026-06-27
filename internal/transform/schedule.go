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

type SessionViewModel struct {
	Begin                 string        `json:"begin"`
	BeginDisplay          string        `json:"beginDisplay"`
	BeginIso              string        `json:"beginIso"`
	BeginTimestampSeconds int64         `json:"beginTimestampSeconds"`
	Color                 string        `json:"color"`
	ContentEntity         *ContentModel `json:"contentEntity"`
	ContentID             int           `json:"contentId"`
	End                   string        `json:"end"`
	EndDisplay            string        `json:"endDisplay"`
	EndIso                string        `json:"endIso"`
	EndTimestampSeconds   int64         `json:"endTimestampSeconds"`
	ID                    int           `json:"id"`
	LocationName          string        `json:"locationName"`
	Session               SessionModel  `json:"session"`
	PeopleText            *string       `json:"speakers"`
	Tags                  []CompactTag  `json:"tags"`
	Title                 string        `json:"title"`
}

type ScheduleDay struct {
	Day      string             `json:"day"`
	Sessions []SessionViewModel `json:"sessions"`
}

type LocationCard struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	ParentID  *int    `json:"parentId"`
	ShortName *string `json:"shortName"`
}

type TagDetail struct {
	Days []ScheduleDay `json:"days"`
	Tag  TagModel      `json:"tag"`
}

type LocationDetail struct {
	Days     []ScheduleDay `json:"days"`
	Location LocationModel `json:"location"`
}

type SessionDetail struct {
	Content  ContentModel   `json:"content"`
	Location *LocationModel `json:"location"`
	People   []PersonModel  `json:"people"`
	Session  SessionModel   `json:"session"`
	Tags     []TagModel     `json:"tags"`
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

func buildScheduleSessionViewModel(session SessionModel, st *stores) SessionViewModel {
	peopleNames := []string{}
	for _, personID := range session.PersonIDs {
		if person, ok := st.peopleByID[personID]; ok && person.Name != "" {
			peopleNames = append(peopleNames, person.Name)
		}
	}

	var contentEntity *ContentModel
	if content, ok := st.contentByID[session.ContentID]; ok {
		contentEntity = &content
	}

	locationName := "Unknown location"
	if session.LocationID != nil {
		if location, ok := st.locationsByID[*session.LocationID]; ok && location.Name != "" {
			locationName = location.Name
		}
	}

	tags := []CompactTag{}
	for _, tagID := range session.TagIDs {
		if tag, ok := st.tagsByID[tagID]; ok {
			tags = append(tags, compactTag(tag))
		}
	}

	var peopleText *string
	if len(peopleNames) > 0 {
		text := strings.Join(peopleNames, ", ")
		peopleText = &text
	}

	return SessionViewModel{
		Begin:                 session.Begin,
		BeginDisplay:          firstNonEmpty(session.BeginDisplay, sessionTimeTable(session.Begin, true, "")),
		BeginIso:              firstNonEmpty(session.BeginIso, isoTime(session.Begin)),
		BeginTimestampSeconds: session.BeginTimestampSeconds,
		Color:                 session.Color,
		ContentEntity:         contentEntity,
		ContentID:             session.ContentID,
		End:                   session.End,
		EndDisplay:            firstNonEmpty(session.EndDisplay, sessionTimeTable(session.End, false, "")),
		EndIso:                firstNonEmpty(session.EndIso, isoTime(session.End)),
		EndTimestampSeconds:   session.EndTimestampSeconds,
		ID:                    session.ID,
		LocationName:          locationName,
		Session:               session,
		PeopleText:            peopleText,
		Tags:                  tags,
		Title:                 session.Title,
	}
}

func buildPageReadyArtifacts(st *stores, indexes builtIndexes, timezone string) (map[string]any, map[string]map[int]any) {
	allSessions := make([]SessionModel, 0, len(st.sessionIDs))
	modelsBySessionID := map[int]SessionViewModel{}
	for _, sessionID := range st.sessionIDs {
		session := st.sessionsByID[sessionID]
		allSessions = append(allSessions, session)
		modelsBySessionID[sessionID] = buildScheduleSessionViewModel(session, st)
	}

	scheduleDays := buildAllScheduleDays(st, indexes, modelsBySessionID, timezone)
	bookmarkSessionsByID := map[string]SessionViewModel{}
	for _, sessionID := range st.sessionIDs {
		if model, ok := modelsBySessionID[sessionID]; ok {
			bookmarkSessionsByID[idKey(sessionID)] = model
		}
	}

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
		"sessions":      {},
		"people":        {},
		"tags":          {},
		"locations":     {},
		"documents":     {},
		"organizations": {},
	}
	for _, contentID := range st.contentIDs {
		details["content"][contentID] = buildContentDetail(st.contentByID[contentID], st, allSessions)
	}
	for _, sessionID := range st.sessionIDs {
		details["sessions"][sessionID] = buildSessionDetail(st.sessionsByID[sessionID], st)
	}
	for _, personID := range st.peopleIDs {
		details["people"][personID] = buildPersonDetail(st.peopleByID[personID], st, allSessions)
	}
	for _, tagID := range st.tagIDs {
		details["tags"][tagID] = TagDetail{
			Days: buildScheduleDaysFromSessions(sessionsFromIDs(sessionIDsForTag(tagID, st, indexes), st.sessionsByID), modelsBySessionID, timezone),
			Tag:  st.tagsByID[tagID],
		}
	}
	for _, locationID := range st.locationIDs {
		location := st.locationsByID[locationID]
		sessions := []SessionModel{}
		for _, session := range allSessions {
			if session.LocationID != nil && *session.LocationID == locationID {
				sessions = append(sessions, session)
			}
		}
		details["locations"][locationID] = LocationDetail{
			Days:     buildScheduleDaysFromSessions(sessions, modelsBySessionID, timezone),
			Location: location,
		}
	}
	for _, id := range st.documentIDs {
		details["documents"][id] = st.documentsByID[id]
	}
	for _, id := range st.organizationIDs {
		details["organizations"][id] = st.organizationsByID[id]
	}

	return map[string]any{
		"announcementsList":    announcements,
		"bookmarkSessionsById": bookmarkSessionsByID,
		"locationCards":        locationCards,
		"scheduleDays":         scheduleDays,
	}, details
}

func buildAllScheduleDays(st *stores, indexes builtIndexes, modelsBySessionID map[int]SessionViewModel, timezone string) []ScheduleDay {
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

func buildScheduleDaysFromSessions(sessions []SessionModel, modelsBySessionID map[int]SessionViewModel, timezone string) []ScheduleDay {
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

func modelsForSessions(sessions []SessionModel, modelsBySessionID map[int]SessionViewModel) []SessionViewModel {
	sortSessions(sessions)
	models := []SessionViewModel{}
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

func sessionIDsForTag(tagID int, st *stores, indexes builtIndexes) []int {
	if indexed, ok := indexes.sessionsByTag[idKey(tagID)]; ok {
		return indexed
	}
	out := []int{}
	for _, sessionID := range st.sessionIDs {
		for _, id := range st.sessionsByID[sessionID].TagIDs {
			if id == tagID {
				out = append(out, sessionID)
				break
			}
		}
	}
	return out
}

func buildSessionDetail(session SessionModel, st *stores) SessionDetail {
	content := st.contentByID[session.ContentID]
	var location *LocationModel
	if session.LocationID != nil {
		if model, ok := st.locationsByID[*session.LocationID]; ok {
			location = &model
		}
	}
	return SessionDetail{
		Content:  content,
		Location: location,
		People:   peopleForIDs(session.PersonIDs, st),
		Session:  session,
		Tags:     tagsForIDs(session.TagIDs, st.tagsByID),
	}
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
