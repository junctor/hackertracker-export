package transform

import (
	"cmp"
	"slices"
)

func buildContentDetail(content map[string]any, st *stores, allEvents []map[string]any) map[string]any {
	sessionIDs := intSlice(content["sessions"])
	sessions := []map[string]any{}
	if len(sessionIDs) > 0 {
		for _, eventID := range sessionIDs {
			if event := st.eventsByID[eventID]; event != nil {
				sessions = append(sessions, event)
			}
		}
	} else {
		contentID := intValue(content["id"])
		for _, event := range allEvents {
			if intValue(event["contentId"]) == contentID {
				sessions = append(sessions, event)
			}
		}
	}
	sortEvents(sessions)
	people := []any{}
	if entries, ok := content["people"].([]any); ok && len(entries) > 0 {
		slices.SortFunc(entries, func(left, right any) int {
			a := left.(map[string]any)
			b := right.(map[string]any)
			ao := nullableOrderValue(a["sortOrder"])
			bo := nullableOrderValue(b["sortOrder"])
			if (ao == nil) != (bo == nil) {
				if ao != nil {
					return -1
				}
				return 1
			}
			if ao != nil && bo != nil && *ao != *bo {
				return cmp.Compare(*ao, *bo)
			}
			return cmp.Compare(intValue(a["personId"]), intValue(b["personId"]))
		})
		for _, entry := range entries {
			personID := intValue(entry.(map[string]any)["personId"])
			if person := st.peopleByID[personID]; person != nil {
				people = append(people, person)
			}
		}
	} else {
		seen := map[int]bool{}
		for _, session := range sessions {
			for _, personID := range append(intSlice(session["speakerIds"]), intSlice(session["personIds"])...) {
				if seen[personID] {
					continue
				}
				if person := st.peopleByID[personID]; person != nil {
					seen[personID] = true
					people = append(people, person)
				}
			}
		}
	}
	return map[string]any{
		"content":   content,
		"locations": uniqueLocationsForEvents(sessions, st),
		"people":    people,
		"sessions":  eventsAny(sessions),
		"tags":      entitiesForIDs(intSlice(content["tagIds"]), st.tagsByID),
	}
}

func buildPersonDetail(person map[string]any, st *stores, allEvents []map[string]any) map[string]any {
	events := []map[string]any{}
	seen := map[int]bool{}
	for _, contentID := range intSlice(person["contentIds"]) {
		content := st.contentByID[contentID]
		for _, eventID := range intSlice(content["sessions"]) {
			if event := st.eventsByID[eventID]; event != nil && !seen[eventID] {
				seen[eventID] = true
				events = append(events, event)
			}
		}
		for _, event := range allEvents {
			eventID := intValue(event["id"])
			if intValue(event["contentId"]) == contentID && !seen[eventID] {
				seen[eventID] = true
				events = append(events, event)
			}
		}
	}
	personID := intValue(person["id"])
	for _, event := range allEvents {
		eventID := intValue(event["id"])
		if seen[eventID] {
			continue
		}
		for _, eventPersonID := range append(intSlice(event["speakerIds"]), intSlice(event["personIds"])...) {
			if eventPersonID == personID {
				seen[eventID] = true
				events = append(events, event)
				break
			}
		}
	}
	sortEvents(events)
	return map[string]any{
		"events":    eventsAny(events),
		"locations": uniqueLocationsForEvents(events, st),
		"person":    person,
	}
}

func uniqueLocationsForEvents(events []map[string]any, st *stores) []any {
	seen := map[int]bool{}
	locations := []any{}
	for _, event := range events {
		locationID := intValue(event["locationId"])
		if locationID == 0 || seen[locationID] {
			continue
		}
		if location := st.locationsByID[locationID]; location != nil {
			seen[locationID] = true
			locations = append(locations, location)
		}
	}
	return locations
}

func entitiesForIDs(ids []int, byID map[int]map[string]any) []any {
	seen := map[int]bool{}
	out := []any{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		if entity := byID[id]; entity != nil {
			seen[id] = true
			out = append(out, entity)
		}
	}
	return out
}
