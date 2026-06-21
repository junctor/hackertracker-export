package transform

import "slices"

type ContentDetail struct {
	Content   ContentModel    `json:"content"`
	Locations []LocationModel `json:"locations"`
	People    []PersonModel   `json:"people"`
	Sessions  []SessionModel  `json:"sessions"`
	Tags      []TagModel      `json:"tags"`
}

type PersonDetail struct {
	Locations []LocationModel `json:"locations"`
	Person    PersonModel     `json:"person"`
	Sessions  []SessionModel  `json:"sessions"`
}

func buildContentDetail(content ContentModel, st *stores, allSessions []SessionModel) ContentDetail {
	sessions := []SessionModel{}
	if len(content.Sessions) > 0 {
		for _, sessionID := range content.Sessions {
			if session, ok := st.sessionsByID[sessionID]; ok {
				sessions = append(sessions, session)
			}
		}
	} else {
		for _, session := range allSessions {
			if session.ContentID == content.ID {
				sessions = append(sessions, session)
			}
		}
	}
	sortSessions(sessions)

	people := []PersonModel{}
	if len(content.People) > 0 {
		peopleEntries := slices.Clone(content.People)
		slices.SortFunc(peopleEntries, compareContentPeople)
		for _, entry := range peopleEntries {
			if person, ok := st.peopleByID[entry.PersonID]; ok {
				people = append(people, person)
			}
		}
	} else {
		seen := map[int]bool{}
		for _, session := range sessions {
			for _, personID := range session.PersonIDs {
				if seen[personID] {
					continue
				}
				if person, ok := st.peopleByID[personID]; ok {
					seen[personID] = true
					people = append(people, person)
				}
			}
		}
	}

	return ContentDetail{
		Content:   content,
		Locations: uniqueLocationsForSessions(sessions, st),
		People:    people,
		Sessions:  sessions,
		Tags:      tagsForIDs(content.TagIDs, st.tagsByID),
	}
}

func buildPersonDetail(person PersonModel, st *stores, allSessions []SessionModel) PersonDetail {
	sessions := []SessionModel{}
	seen := map[int]bool{}
	for _, contentID := range person.ContentIDs {
		content, ok := st.contentByID[contentID]
		if !ok {
			continue
		}
		for _, sessionID := range content.Sessions {
			if session, ok := st.sessionsByID[sessionID]; ok && !seen[sessionID] {
				seen[sessionID] = true
				sessions = append(sessions, session)
			}
		}
		for _, session := range allSessions {
			if session.ContentID == contentID && !seen[session.ID] {
				seen[session.ID] = true
				sessions = append(sessions, session)
			}
		}
	}

	for _, session := range allSessions {
		if seen[session.ID] {
			continue
		}
		for _, sessionPersonID := range session.PersonIDs {
			if sessionPersonID == person.ID {
				seen[session.ID] = true
				sessions = append(sessions, session)
				break
			}
		}
	}
	sortSessions(sessions)

	return PersonDetail{
		Locations: uniqueLocationsForSessions(sessions, st),
		Person:    person,
		Sessions:  sessions,
	}
}

func uniqueLocationsForSessions(sessions []SessionModel, st *stores) []LocationModel {
	seen := map[int]bool{}
	locations := []LocationModel{}
	for _, session := range sessions {
		if session.LocationID == nil || seen[*session.LocationID] {
			continue
		}
		if location, ok := st.locationsByID[*session.LocationID]; ok {
			seen[*session.LocationID] = true
			locations = append(locations, location)
		}
	}
	return locations
}

func tagsForIDs(ids []int, byID map[int]TagModel) []TagModel {
	seen := map[int]bool{}
	out := []TagModel{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		if entity, ok := byID[id]; ok {
			seen[id] = true
			out = append(out, entity)
		}
	}
	return out
}

func peopleForIDs(ids []int, st *stores) []PersonModel {
	seen := map[int]bool{}
	out := []PersonModel{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		if person, ok := st.peopleByID[id]; ok {
			seen[id] = true
			out = append(out, person)
		}
	}
	return out
}
