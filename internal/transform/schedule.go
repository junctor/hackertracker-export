package transform

import (
	"sort"

	"github.com/junctor/hackertracker-export/internal/export"
)

type builtIndexes struct {
	eventsByDay map[string][]int
	eventsByTag map[string][]int
}

func buildIndexes(st *stores, timezone string) builtIndexes {
	eventsByDay := map[string][]int{}
	eventsByTag := map[string][]int{}
	eventStarts := map[int]int64{}

	for _, eventID := range st.eventIDs {
		event := st.eventsByID[eventID]
		eventStarts[eventID] = export.TimestampSeconds(stringValue(event["begin"]))
		export.AddToStringIndex(eventsByDay, export.EventDay(stringValue(event["begin"]), timezone), eventID)
		for _, tagID := range intSlice(event["tagIds"]) {
			export.AddToStringIndex(eventsByTag, stringValue(tagID), eventID)
		}
	}
	export.SortEventIndex(eventsByDay, eventStarts)
	export.SortEventIndex(eventsByTag, eventStarts)
	return builtIndexes{eventsByDay: eventsByDay, eventsByTag: eventsByTag}
}

func buildScheduleEventViewModel(event map[string]any, st *stores, timezone string) map[string]any {
	speakerIDs := intSlice(event["speakerIds"])
	if len(speakerIDs) == 0 {
		speakerIDs = intSlice(event["personIds"])
	}
	speakerNames := []string{}
	for _, personID := range speakerIDs {
		if person := st.peopleByID[personID]; person != nil && person["name"] != nil {
			speakerNames = append(speakerNames, stringValue(person["name"]))
		}
	}
	contentID, _ := export.NormalizeID(event["contentId"])
	var contentEntity any
	if contentID != 0 {
		contentEntity = st.contentByID[contentID]
	}
	locationName := "Unknown location"
	if locationID, ok := export.NormalizeID(event["locationId"]); ok {
		if loc := st.locationsByID[locationID]; loc != nil && loc["name"] != nil {
			locationName = stringValue(loc["name"])
		}
	}
	tags := []any{}
	for _, tagID := range intSlice(event["tagIds"]) {
		if tag := st.tagsByID[tagID]; tag != nil {
			tags = append(tags, compactTag(tag))
		}
	}
	var speakers any
	if len(speakerNames) > 0 {
		speakers = joinComma(speakerNames)
	}
	begin := stringValue(event["begin"])
	end := stringValue(event["end"])
	return map[string]any{
		"begin":                 begin,
		"beginDisplay":          nonEmptyString(event["beginDisplay"], export.EventTimeTable(begin, true, timezone)),
		"beginIso":              nonEmptyString(event["beginIso"], export.ISOTime(begin)),
		"beginTimestampSeconds": export.TimestampSeconds(begin),
		"color":                 nonEmptyString(event["color"], ""),
		"contentEntity":         contentEntity,
		"contentId":             event["contentId"],
		"end":                   end,
		"endDisplay":            nonEmptyString(event["endDisplay"], export.EventTimeTable(end, false, timezone)),
		"endIso":                nonEmptyString(event["endIso"], export.ISOTime(end)),
		"endTimestampSeconds":   export.TimestampSeconds(end),
		"id":                    event["id"],
		"locationName":          locationName,
		"session":               event,
		"speakers":              speakers,
		"tags":                  tags,
		"title":                 event["title"],
	}
}

func buildPageReadyArtifacts(st *stores, indexes builtIndexes, timezone string) (map[string]any, map[string]map[int]any) {
	allEvents := make([]map[string]any, 0, len(st.eventIDs))
	modelsByEventID := map[int]map[string]any{}
	for _, eventID := range st.eventIDs {
		event := st.eventsByID[eventID]
		allEvents = append(allEvents, event)
		modelsByEventID[eventID] = buildScheduleEventViewModel(event, st, timezone)
	}
	scheduleDays := buildAllScheduleDays(st, indexes, modelsByEventID, timezone)
	bookmarkEventsByID := map[string]any{}
	for _, eventID := range st.eventIDs {
		if model := modelsByEventID[eventID]; model != nil {
			bookmarkEventsByID[stringValue(eventID)] = model
		}
	}
	locationCards := []any{}
	for _, locationID := range st.locationIDs {
		location := st.locationsByID[locationID]
		locationCards = append(locationCards, map[string]any{
			"id":        location["id"],
			"name":      location["name"],
			"parentId":  location["parentId"],
			"shortName": location["shortName"],
		})
	}
	announcements := []map[string]any{}
	for _, articleID := range st.articleIDs {
		announcements = append(announcements, st.articlesByID[articleID])
	}
	sort.Slice(announcements, func(i, j int) bool {
		ai := int64Value(announcements[i]["updatedAtMs"])
		aj := int64Value(announcements[j]["updatedAtMs"])
		if ai != aj {
			return ai > aj
		}
		return intValue(announcements[i]["id"]) < intValue(announcements[j]["id"])
	})

	details := map[string]map[int]any{
		"content":       {},
		"people":        {},
		"tags":          {},
		"locations":     {},
		"documents":     {},
		"organizations": {},
	}
	for _, contentID := range st.contentIDs {
		details["content"][contentID] = buildContentDetail(st.contentByID[contentID], st, allEvents)
	}
	for _, personID := range st.peopleIDs {
		details["people"][personID] = buildPersonDetail(st.peopleByID[personID], st, allEvents)
	}
	for _, tagID := range st.tagIDs {
		tag := st.tagsByID[tagID]
		details["tags"][tagID] = map[string]any{
			"days": buildScheduleDaysFromEvents(eventsFromIDs(eventIDsForTag(tagID, st, indexes), st.eventsByID), modelsByEventID, timezone),
			"tag":  tag,
		}
	}
	for _, locationID := range st.locationIDs {
		location := st.locationsByID[locationID]
		events := []map[string]any{}
		for _, event := range allEvents {
			if intValue(event["locationId"]) == locationID {
				events = append(events, event)
			}
		}
		details["locations"][locationID] = map[string]any{
			"days":     buildScheduleDaysFromEvents(events, modelsByEventID, timezone),
			"location": location,
		}
	}
	for _, id := range st.documentIDs {
		details["documents"][id] = st.documentsByID[id]
	}
	for _, id := range st.organizationIDs {
		details["organizations"][id] = st.organizationsByID[id]
	}
	return map[string]any{
		"announcementsList":  eventsAny(announcements),
		"bookmarkEventsById": bookmarkEventsByID,
		"locationCards":      locationCards,
		"scheduleDays":       scheduleDays,
	}, details
}

func buildAllScheduleDays(st *stores, indexes builtIndexes, modelsByEventID map[int]map[string]any, timezone string) []any {
	keys := make([]string, 0, len(indexes.eventsByDay))
	for day := range indexes.eventsByDay {
		keys = append(keys, day)
	}
	sort.Strings(keys)
	days := []any{}
	for _, day := range keys {
		events := eventsFromIDs(indexes.eventsByDay[day], st.eventsByID)
		models := modelsForEvents(events, modelsByEventID)
		if len(models) > 0 {
			days = append(days, map[string]any{"day": day, "events": models})
		}
	}
	if len(keys) > 0 {
		return days
	}
	return buildScheduleDaysFromEvents(eventsFromIDs(st.eventIDs, st.eventsByID), modelsByEventID, timezone)
}

func buildScheduleDaysFromEvents(events []map[string]any, modelsByEventID map[int]map[string]any, timezone string) []any {
	groups := map[string][]map[string]any{}
	for _, event := range events {
		day := export.EventDay(stringValue(event["begin"]), timezone)
		if day == "" {
			continue
		}
		groups[day] = append(groups[day], event)
	}
	keys := make([]string, 0, len(groups))
	for day := range groups {
		keys = append(keys, day)
	}
	sort.Strings(keys)
	out := []any{}
	for _, day := range keys {
		sortEvents(groups[day])
		models := modelsForEvents(groups[day], modelsByEventID)
		if len(models) > 0 {
			out = append(out, map[string]any{"day": day, "events": models})
		}
	}
	return out
}

func modelsForEvents(events []map[string]any, modelsByEventID map[int]map[string]any) []any {
	sortEvents(events)
	models := []any{}
	for _, event := range events {
		if model := modelsByEventID[intValue(event["id"])]; model != nil {
			models = append(models, model)
		}
	}
	return models
}

func sortEvents(events []map[string]any) {
	sort.Slice(events, func(i, j int) bool {
		ai := export.TimestampSeconds(stringValue(events[i]["begin"]))
		aj := export.TimestampSeconds(stringValue(events[j]["begin"]))
		if ai != aj {
			return ai < aj
		}
		return intValue(events[i]["id"]) < intValue(events[j]["id"])
	})
}

func eventsFromIDs(ids []int, byID map[int]map[string]any) []map[string]any {
	events := []map[string]any{}
	for _, id := range ids {
		if event := byID[id]; event != nil {
			events = append(events, event)
		}
	}
	return events
}

func eventIDsForTag(tagID int, st *stores, indexes builtIndexes) []int {
	if indexed, ok := indexes.eventsByTag[stringValue(tagID)]; ok {
		return indexed
	}
	out := []int{}
	for _, eventID := range st.eventIDs {
		for _, id := range intSlice(st.eventsByID[eventID]["tagIds"]) {
			if id == tagID {
				out = append(out, eventID)
				break
			}
		}
	}
	return out
}
