package transform

import "slices"

type ContentDetailContent struct {
	Color       string      `json:"color,omitempty"`
	Description string      `json:"description,omitempty"`
	ID          int         `json:"id"`
	Links       []LinkModel `json:"links,omitempty"`
	LogoURL     string      `json:"logoUrl,omitempty"`
	Title       string      `json:"title"`
}

type DetailPersonSummary struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type DetailSession struct {
	Begin        string `json:"begin"`
	Color        string `json:"color,omitempty"`
	ContentID    int    `json:"contentId"`
	End          string `json:"end"`
	ID           int    `json:"id"`
	LocationName string `json:"locationName,omitempty"`
	Title        string `json:"title"`
}

type PersonDetailPerson struct {
	Affiliations []string    `json:"affiliations,omitempty"`
	AvatarURL    string      `json:"avatarUrl,omitempty"`
	Description  string      `json:"description,omitempty"`
	ID           int         `json:"id"`
	Links        []LinkModel `json:"links,omitempty"`
	Name         string      `json:"name"`
	Pronouns     string      `json:"pronouns,omitempty"`
}

type ContentDetail struct {
	AccentColor    string                `json:"accentColor,omitempty"`
	Content        ContentDetailContent  `json:"content"`
	People         []DetailPersonSummary `json:"people"`
	RelatedContent []ContentCard         `json:"relatedContent"`
	Sessions       []DetailSession       `json:"sessions"`
	Tags           []CompactTag          `json:"tags"`
}

type PersonDetail struct {
	Person   PersonDetailPerson `json:"person"`
	Sessions []DetailSession    `json:"sessions"`
}

func buildContentDetail(content ContentModel, st *stores, allSessions []SessionModel) ContentDetail {
	sessions := sessionsForContent(content, allSessions, st)
	people := peopleForContent(content, sessions, st)
	compactTags := sortedCompactTags(content.TagIDs, st.tagsByID)

	return ContentDetail{
		AccentColor: contentDetailAccentColor(content, sessions, st),
		Content: ContentDetailContent{
			Color:       content.Color,
			Description: content.Description,
			ID:          content.ID,
			Links:       content.Links,
			LogoURL:     content.LogoURL,
			Title:       content.Title,
		},
		People:         people,
		RelatedContent: relatedContentCards(content, st),
		Sessions:       detailSessions(sessions, st),
		Tags:           compactTags,
	}
}

func sessionsForContent(content ContentModel, allSessions []SessionModel, st *stores) []SessionModel {
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
	return sessions
}

func peopleForContent(content ContentModel, sessions []SessionModel, st *stores) []DetailPersonSummary {
	people := []DetailPersonSummary{}
	appendPerson := func(personID int) {
		if person, ok := st.peopleByID[personID]; ok {
			people = append(people, DetailPersonSummary{ID: person.ID, Name: person.Name})
		}
	}
	if len(content.People) > 0 {
		entries := slices.Clone(content.People)
		slices.SortFunc(entries, compareContentPeople)
		for _, entry := range entries {
			appendPerson(entry.PersonID)
		}
		return people
	}

	seen := map[int]bool{}
	for _, session := range sessions {
		for _, personID := range session.PersonIDs {
			if seen[personID] {
				continue
			}
			seen[personID] = true
			appendPerson(personID)
		}
	}
	return people
}

func contentDetailAccentColor(content ContentModel, sessions []SessionModel, st *stores) string {
	if content.Color != "" {
		return content.Color
	}
	if len(sessions) == 0 {
		return ""
	}
	if sessions[0].Color != "" {
		return sessions[0].Color
	}
	if len(sessions[0].TagIDs) > 0 {
		return st.tagsByID[sessions[0].TagIDs[0]].ColorBackground
	}
	return ""
}

func detailSessions(sessions []SessionModel, st *stores) []DetailSession {
	out := make([]DetailSession, 0, len(sessions))
	for _, session := range sessions {
		locationName := ""
		if session.LocationID != nil {
			locationName = st.locationsByID[*session.LocationID].Name
		}
		out = append(out, DetailSession{
			Begin:        session.Begin,
			Color:        session.Color,
			ContentID:    session.ContentID,
			End:          session.End,
			ID:           session.ID,
			LocationName: locationName,
			Title:        session.Title,
		})
	}
	return out
}

func relatedContentCards(content ContentModel, st *stores) []ContentCard {
	seen := map[int]bool{}
	cards := []ContentCard{}
	for _, relatedID := range content.RelatedContentIDs {
		if relatedID == content.ID || seen[relatedID] {
			continue
		}
		related, ok := st.contentByID[relatedID]
		if !ok {
			continue
		}
		seen[relatedID] = true
		cards = append(cards, buildContentCard(related, st))
	}
	return cards
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
		Person: PersonDetailPerson{
			Affiliations: person.Affiliations,
			AvatarURL:    person.AvatarURL,
			Description:  person.Description,
			ID:           person.ID,
			Links:        person.Links,
			Name:         person.Name,
			Pronouns:     person.Pronouns,
		},
		Sessions: detailSessions(sessions, st),
	}
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
