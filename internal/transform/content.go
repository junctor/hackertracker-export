package transform

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/junctor/hackertracker-export/internal/export"
	"github.com/junctor/hackertracker-export/pkg/hackertracker"
)

type stores struct {
	entities map[string]any

	eventsByID        map[int]map[string]any
	contentByID       map[int]map[string]any
	peopleByID        map[int]map[string]any
	locationsByID     map[int]map[string]any
	organizationsByID map[int]map[string]any
	tagsByID          map[int]map[string]any
	tagTypesByID      map[int]map[string]any
	documentsByID     map[int]map[string]any
	articlesByID      map[int]map[string]any

	eventIDs        []int
	contentIDs      []int
	peopleIDs       []int
	locationIDs     []int
	organizationIDs []int
	tagIDs          []int
	tagTypeIDs      []int
	documentIDs     []int
	articleIDs      []int
}

func Build(conf hackertracker.Conference, data hackertracker.SourceData) (export.Artifacts, error) {
	if conf.Timezone == "" {
		return export.Artifacts{}, fmt.Errorf("missing conference timezone")
	}

	st, err := buildEntities(data, conf.Timezone)
	if err != nil {
		return export.Artifacts{}, err
	}
	indexes := buildIndexes(st, conf.Timezone)
	pageViews, details := buildPageReadyArtifacts(st, indexes, conf.Timezone)
	views := buildViews(st)
	for key, value := range pageViews {
		views[key] = value
	}

	return export.Artifacts{
		Manifest: map[string]any{
			"buildTimestamp": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
			"code":           conf.Code,
			"name":           conf.Name,
			"schemaVersion":  2,
			"timezone":       conf.Timezone,
		},
		Entities: st.entities,
		Indexes:  map[string]any{"eventsByDay": indexes.eventsByDay, "eventsByTag": indexes.eventsByTag},
		Views:    views,
		Derived:  map[string]any{"tagIdsByLabel": buildTagIDsByLabel(data.TagTypes)},
		Details:  details,
	}, nil
}

func buildEntities(data hackertracker.SourceData, timezone string) (*stores, error) {
	refs := struct {
		locationIDs map[int]bool
		personIDs   map[int]bool
		tagIDs      map[int]bool
		contentIDs  map[int]bool
	}{
		locationIDs: map[int]bool{},
		personIDs:   map[int]bool{},
		tagIDs:      map[int]bool{},
		contentIDs:  map[int]bool{},
	}
	for _, loc := range data.Locations {
		if id, ok := normalizeID(loc.ID); ok {
			refs.locationIDs[id] = true
		}
	}
	for _, person := range data.Speakers {
		if id, ok := normalizeID(person.ID); ok {
			refs.personIDs[id] = true
		}
	}
	for _, group := range data.TagTypes {
		for _, tag := range group.Tags {
			if id, ok := normalizeID(tag.ID); ok {
				refs.tagIDs[id] = true
			}
		}
	}
	for _, item := range data.Content {
		if id, ok := normalizeID(item.ID); ok {
			refs.contentIDs[id] = true
		}
	}

	st := &stores{
		entities:          map[string]any{},
		eventsByID:        map[int]map[string]any{},
		contentByID:       map[int]map[string]any{},
		peopleByID:        map[int]map[string]any{},
		locationsByID:     map[int]map[string]any{},
		organizationsByID: map[int]map[string]any{},
		tagsByID:          map[int]map[string]any{},
		tagTypesByID:      map[int]map[string]any{},
		documentsByID:     map[int]map[string]any{},
		articlesByID:      map[int]map[string]any{},
	}

	contentEventSources := buildEventSources(data)
	for _, event := range contentEventSources {
		model, err := buildEventModel(event, refs.locationIDs, refs.personIDs, refs.tagIDs, refs.contentIDs, timezone)
		if err != nil {
			return nil, err
		}
		putEntity(st.eventsByID, &st.eventIDs, model)
	}

	for _, item := range data.Content {
		model, err := buildContentModel(item, refs.personIDs, refs.tagIDs, refs.contentIDs)
		if err != nil {
			return nil, err
		}
		putEntity(st.contentByID, &st.contentIDs, model)
	}
	for _, person := range data.Speakers {
		model, err := buildPersonModel(person, refs.contentIDs)
		if err != nil {
			return nil, err
		}
		putEntity(st.peopleByID, &st.peopleIDs, model)
	}
	for _, loc := range data.Locations {
		id, ok := normalizeID(loc.ID)
		if !ok {
			return nil, fmt.Errorf("location missing id")
		}
		model := map[string]any{"id": id, "name": loc.Name, "shortName": nil, "parentId": nil}
		if loc.ShortName != "" {
			model["shortName"] = loc.ShortName
		}
		if parentID, ok := normalizeID(loc.ParentID); ok {
			model["parentId"] = parentID
		}
		putEntity(st.locationsByID, &st.locationIDs, model)
	}
	for _, org := range data.Organizations {
		model, err := buildOrganizationModel(org, refs.tagIDs)
		if err != nil {
			return nil, err
		}
		putEntity(st.organizationsByID, &st.organizationIDs, model)
	}
	tags := buildTags(data.TagTypes)
	for _, tag := range tags {
		putEntity(st.tagsByID, &st.tagIDs, tag)
	}
	for _, tagType := range data.TagTypes {
		id, ok := normalizeID(tagType.ID)
		if !ok {
			return nil, fmt.Errorf("tag type missing id")
		}
		model := map[string]any{
			"category":    nullableString(tagType.Category),
			"id":          id,
			"isBrowsable": tagType.IsBrowsable,
			"label":       tagType.Label,
			"sortOrder":   nullableInt(tagType.SortOrder),
		}
		putEntity(st.tagTypesByID, &st.tagTypeIDs, model)
	}
	for _, article := range data.Articles {
		id, ok := normalizeID(article.ID)
		if !ok {
			return nil, fmt.Errorf("article missing id")
		}
		model := map[string]any{
			"id":   id,
			"name": article.Name,
		}
		if article.Text != nil {
			model["text"] = *article.Text
		} else {
			model["text"] = nil
		}
		if updated := resolveUpdatedAtMs(article.UpdatedAt, article.UpdatedTSZ, article.UpdatedAtStr); updated != nil {
			model["updatedAtMs"] = *updated
		} else {
			model["updatedAtMs"] = nil
		}
		putEntity(st.articlesByID, &st.articleIDs, model)
	}
	for _, doc := range data.Documents {
		id, ok := normalizeID(doc.ID)
		if !ok {
			return nil, fmt.Errorf("document missing id")
		}
		model := map[string]any{"bodyText": nullableString(doc.BodyText), "id": id, "titleText": nullableString(doc.TitleText)}
		if updated := resolveUpdatedAtMs(doc.UpdatedAt, doc.UpdatedTSZ, doc.UpdatedAtStr); updated != nil {
			model["updatedAtMs"] = *updated
		}
		putEntity(st.documentsByID, &st.documentIDs, model)
	}

	st.entities["events"] = entityStore(st.eventIDs, st.eventsByID)
	st.entities["content"] = entityStore(st.contentIDs, st.contentByID)
	st.entities["people"] = entityStore(st.peopleIDs, st.peopleByID)
	st.entities["locations"] = entityStore(st.locationIDs, st.locationsByID)
	st.entities["organizations"] = entityStore(st.organizationIDs, st.organizationsByID)
	st.entities["tags"] = entityStore(st.tagIDs, st.tagsByID)
	st.entities["tagTypes"] = entityStore(st.tagTypeIDs, st.tagTypesByID)
	st.entities["articles"] = entityStore(st.articleIDs, st.articlesByID)
	st.entities["documents"] = entityStore(st.documentIDs, st.documentsByID)
	return st, nil
}

func buildEventModel(event hackertracker.Event, locationIDs, personIDs, tagIDs, contentIDs map[int]bool, timezone string) (map[string]any, error) {
	id, ok := normalizeID(event.ID)
	if !ok {
		return nil, fmt.Errorf("event missing id")
	}
	speakerRaw := make([]any, 0, len(event.Speakers))
	for _, speaker := range event.Speakers {
		speakerRaw = append(speakerRaw, speaker.ID)
	}
	personRaw := make([]any, 0, len(event.People))
	for _, person := range event.People {
		personRaw = append(personRaw, person.PersonID)
	}
	speakerIDs := uniqueIDs(speakerRaw, personIDs)
	personIDsOut := uniqueIDs(personRaw, personIDs)
	tagIDsOut := uniqueIDs(event.TagIDs, tagIDs)
	slices.Sort(speakerIDs)
	slices.Sort(personIDsOut)
	slices.Sort(tagIDsOut)

	locationID := 0
	locationOK := false
	if event.Location != nil {
		locationID, locationOK = normalizeID(event.Location.ID)
	}
	if !locationOK {
		locationID, locationOK = normalizeID(event.LocationID)
	}
	var resolvedLocation any
	if locationOK && locationIDs[locationID] {
		resolvedLocation = locationID
	}
	contentID, ok := normalizeID(event.ContentID)
	var resolvedContent any
	if ok && contentIDs[contentID] {
		resolvedContent = contentID
	}

	model := map[string]any{
		"begin":        event.BeginTSZ,
		"beginDisplay": eventTimeTable(event.BeginTSZ, true, timezone),
		"beginIso":     isoTime(event.BeginTSZ),
		"contentId":    resolvedContent,
		"end":          event.EndTSZ,
		"endDisplay":   eventTimeTable(event.EndTSZ, false, timezone),
		"endIso":       isoTime(event.EndTSZ),
		"id":           id,
		"locationId":   resolvedLocation,
		"title":        event.Title,
	}
	if len(speakerIDs) > 0 {
		model["speakerIds"] = speakerIDs
	}
	if len(personIDsOut) > 0 {
		model["personIds"] = personIDsOut
	}
	if len(tagIDsOut) > 0 {
		model["tagIds"] = tagIDsOut
	}
	if event.Type != nil && event.Type.Color != "" {
		model["color"] = event.Type.Color
	}
	return model, nil
}

func buildContentModel(item hackertracker.Content, personIDs, tagIDs, contentIDs map[int]bool) (map[string]any, error) {
	id, ok := normalizeID(item.ID)
	if !ok {
		return nil, fmt.Errorf("content item missing id")
	}
	tagIDsOut := uniqueIDs(item.TagIDs, tagIDs)
	slices.Sort(tagIDsOut)
	related := uniqueIDs(item.RelatedContentIDs, contentIDs)
	slices.Sort(related)
	sessionRaw := make([]any, 0, len(item.Sessions))
	for _, session := range item.Sessions {
		sessionRaw = append(sessionRaw, session.SessionID)
	}
	sessions := uniqueIDs(sessionRaw, nil)
	slices.Sort(sessions)

	model := map[string]any{"id": id, "relatedContentIds": related, "sessions": sessions, "title": item.Title}
	if item.Description != "" {
		model["description"] = item.Description
	}
	if len(item.Links) > 0 {
		model["links"] = linksToAny(item.Links)
	}
	if len(tagIDsOut) > 0 {
		model["tagIds"] = tagIDsOut
	}

	peopleEntries := []map[string]any{}
	peopleOrder := map[int]*int{}
	for _, person := range item.People {
		personID, ok := normalizeID(person.PersonID)
		if !ok || !personIDs[personID] {
			continue
		}
		order := normalizeOrder(person.SortOrder)
		peopleEntries = append(peopleEntries, map[string]any{"personId": personID, "sortOrder": nullableIntPtr(order)})
		peopleOrder[personID] = order
	}
	seen := map[int]bool{}
	peopleIDs := []int{}
	for _, entry := range peopleEntries {
		personID := entry["personId"].(int)
		if seen[personID] {
			continue
		}
		seen[personID] = true
		peopleIDs = append(peopleIDs, personID)
	}
	slices.SortFunc(peopleIDs, func(aID, bID int) int {
		a := peopleOrder[aID]
		b := peopleOrder[bID]
		if (a == nil) != (b == nil) {
			if a != nil {
				return -1
			}
			return 1
		}
		if a != nil && b != nil && *a != *b {
			return cmp.Compare(*a, *b)
		}
		return cmp.Compare(aID, bID)
	})
	if len(peopleIDs) > 0 {
		people := make([]any, 0, len(peopleIDs))
		for _, personID := range peopleIDs {
			people = append(people, map[string]any{"personId": personID, "sortOrder": nullableIntPtr(peopleOrder[personID])})
		}
		model["people"] = people
	}
	return model, nil
}

func buildPersonModel(person hackertracker.Speaker, contentIDs map[int]bool) (map[string]any, error) {
	id, ok := normalizeID(person.ID)
	if !ok {
		return nil, fmt.Errorf("person missing id")
	}
	contentIDsOut := uniqueIDs(person.ContentIDs, contentIDs)
	slices.Sort(contentIDsOut)
	model := map[string]any{"contentIds": contentIDsOut, "id": id, "name": person.Name}
	if person.Description != "" {
		model["description"] = person.Description
	}
	if person.Pronouns != "" {
		model["pronouns"] = person.Pronouns
	}
	if person.Title != "" {
		model["title"] = person.Title
	}
	if len(person.Affiliations) > 0 {
		items := make([]any, len(person.Affiliations))
		for i, value := range person.Affiliations {
			items[i] = value
		}
		model["affiliations"] = items
	}
	if person.Avatar != nil && person.Avatar.URL != "" {
		model["avatarUrl"] = person.Avatar.URL
	}
	if len(person.Links) > 0 {
		model["links"] = linksToAny(person.Links)
	}
	return model, nil
}

func buildOrganizationModel(org hackertracker.Organization, tagIDs map[int]bool) (map[string]any, error) {
	id, ok := normalizeID(org.ID)
	if !ok {
		return nil, fmt.Errorf("organization missing id")
	}
	tags := uniqueIDs(org.TagIDs, tagIDs)
	slices.Sort(tags)
	var tagIDAsOrganizer any
	if tagID, ok := normalizeID(org.TagIDAsOrganizer); ok {
		tagIDAsOrganizer = tagID
	}
	model := map[string]any{
		"description":      org.Description,
		"id":               id,
		"links":            linksToAny(org.Links),
		"name":             org.Name,
		"tagIdAsOrganizer": tagIDAsOrganizer,
	}
	if org.Logo != nil && org.Logo.URL != "" {
		model["logoUrl"] = org.Logo.URL
	}
	if len(tags) > 0 {
		model["tagIds"] = tags
	}
	return model, nil
}

func buildTags(tagTypes []hackertracker.TagType) []map[string]any {
	tags := []map[string]any{}
	for _, group := range tagTypes {
		typeID, _ := normalizeID(group.ID)
		for _, tag := range group.Tags {
			id, ok := normalizeID(tag.ID)
			if !ok {
				continue
			}
			tags = append(tags, map[string]any{
				"colorBackground": tag.ColorBackground,
				"colorForeground": tag.ColorForeground,
				"id":              id,
				"label":           tag.Label,
				"sortOrder":       nullableInt(tag.SortOrder),
				"tagTypeId":       typeID,
			})
		}
	}
	slices.SortFunc(tags, compareTags)
	return tags
}

func buildEventSources(data hackertracker.SourceData) []hackertracker.Event {
	eventByID := map[int]hackertracker.Event{}
	for _, event := range data.Events {
		if id, ok := normalizeID(event.ID); ok {
			eventByID[id] = event
		}
	}
	sessionEventIDs := map[int]bool{}
	sessionEvents := []hackertracker.Event{}
	for _, content := range data.Content {
		for _, session := range content.Sessions {
			sessionID, ok := normalizeID(session.SessionID)
			if !ok {
				continue
			}
			existing := eventByID[sessionID]
			locationID, locationOK := normalizeID(session.LocationID)
			if !locationOK {
				if existing.Location != nil {
					locationID, locationOK = normalizeID(existing.Location.ID)
				}
				if !locationOK {
					locationID, locationOK = normalizeID(existing.LocationID)
				}
			}
			people := existing.People
			if len(content.People) > 0 {
				people = make([]hackertracker.EventPerson, 0, len(content.People))
				for _, person := range content.People {
					people = append(people, hackertracker.EventPerson{PersonID: person.PersonID})
				}
			}
			speakers := existing.Speakers
			if len(speakers) == 0 && len(people) > 0 {
				for _, person := range people {
					speakers = append(speakers, hackertracker.Ref{ID: person.PersonID})
				}
			}
			event := existing
			event.ID = json.Number(fmt.Sprint(sessionID))
			if content.Title != "" {
				event.Title = content.Title
			}
			if content.ID != "" {
				event.ContentID = content.ID
			}
			if session.BeginTSZ != "" {
				event.BeginTSZ = session.BeginTSZ
			}
			if session.EndTSZ != "" {
				event.EndTSZ = session.EndTSZ
			}
			if locationOK {
				event.LocationID = json.Number(fmt.Sprint(locationID))
				event.Location = &hackertracker.Ref{ID: json.Number(fmt.Sprint(locationID))}
			}
			event.People = people
			event.Speakers = speakers
			if len(content.TagIDs) > 0 {
				event.TagIDs = content.TagIDs
			}
			sessionEventIDs[sessionID] = true
			sessionEvents = append(sessionEvents, event)
		}
	}
	if len(sessionEvents) == 0 {
		return data.Events
	}
	out := slices.Clone(sessionEvents)
	for _, event := range data.Events {
		if id, ok := normalizeID(event.ID); ok && !sessionEventIDs[id] {
			out = append(out, event)
		}
	}
	return out
}

func putEntity(store map[int]map[string]any, ids *[]int, model map[string]any) {
	id := model["id"].(int)
	if _, exists := store[id]; exists {
		return
	}
	store[id] = model
	*ids = append(*ids, id)
}

func entityStore(ids []int, byID map[int]map[string]any) map[string]any {
	slices.Sort(ids)
	byIDOut := map[string]any{}
	for _, id := range ids {
		byIDOut[fmt.Sprint(id)] = byID[id]
	}
	return map[string]any{"allIds": ids, "byId": byIDOut}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value any) any {
	if id, ok := normalizeID(value); ok {
		return id
	}
	return nil
}

func nullableIntPtr(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func linksToAny(links []hackertracker.Link) []any {
	out := make([]any, 0, len(links))
	for _, link := range links {
		item := map[string]any{
			"label": link.Label,
			"type":  link.Type,
			"url":   link.URL,
		}
		for key, value := range item {
			if value == "" {
				delete(item, key)
			}
		}
		out = append(out, item)
	}
	return out
}

func compareTags(a, b map[string]any) int {
	ao, _ := normalizeID(a["sortOrder"])
	bo, _ := normalizeID(b["sortOrder"])
	ai, _ := normalizeID(a["id"])
	bi, _ := normalizeID(b["id"])
	return cmp.Or(
		cmp.Compare(ao, bo),
		alphaCompare(fmt.Sprint(a["label"]), fmt.Sprint(b["label"])),
		cmp.Compare(ai, bi),
	)
}

func alphaCompare(a, b string) int {
	return strings.Compare(alphaSortKey(a), alphaSortKey(b))
}

func alphaSortKey(value string) string {
	rawLower := strings.ToLower(value)
	if rawLower == "" {
		return "8:"
	}
	r, _ := utf8.DecodeRuneInString(rawLower)
	lower := foldSortString(rawLower)
	switch r {
	case '?':
		return "0:" + lower
	case '.':
		return "1:" + lower
	case '"', '“', '”':
		return "2:" + strings.TrimLeft(lower, "\"“”")
	case '[':
		return "3:" + lower
	case '#':
		return "4:" + lower
	case '+':
		return "5:" + lower
	case '$':
		return "6:" + lower
	}
	if unicode.IsNumber(r) {
		return "7:" + lower
	}
	return "8:" + lower
}

func normalizeForSearch(text any) string {
	if text == nil {
		return ""
	}
	lowered := strings.ToLower(fmt.Sprint(text))
	var b strings.Builder
	hadSpace := false
	for _, r := range lowered {
		r = foldLatinAccent(r)
		if unicode.IsMark(r) {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if hadSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			hadSpace = false
			continue
		}
		hadSpace = true
	}
	return strings.TrimSpace(b.String())
}

func foldLatinAccent(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ä', 'ã', 'å', 'ā', 'ă', 'ą':
		return 'a'
	case 'ç', 'ć', 'ĉ', 'ċ', 'č':
		return 'c'
	case 'ď', 'đ':
		return 'd'
	case 'é', 'è', 'ê', 'ë', 'ē', 'ĕ', 'ė', 'ę', 'ě':
		return 'e'
	case 'í', 'ì', 'î', 'ï', 'ĩ', 'ī', 'ĭ', 'į':
		return 'i'
	case 'ñ', 'ń', 'ņ', 'ň':
		return 'n'
	case 'ó', 'ò', 'ô', 'ö', 'õ', 'ō', 'ŏ', 'ő':
		return 'o'
	case 'ŕ', 'ŗ', 'ř':
		return 'r'
	case 'ś', 'ŝ', 'ş', 'š':
		return 's'
	case 'ť', 'ţ', 'ŧ':
		return 't'
	case 'ú', 'ù', 'û', 'ü', 'ũ', 'ū', 'ŭ', 'ů', 'ű', 'ų':
		return 'u'
	case 'ý', 'ÿ', 'ŷ':
		return 'y'
	case 'ź', 'ż', 'ž':
		return 'z'
	case 'æ':
		return 'a'
	case 'œ':
		return 'o'
	case 'ß':
		return 's'
	default:
		return r
	}
}

func foldSortString(value string) string {
	var b strings.Builder
	for _, r := range value {
		if unicode.IsSpace(r) {
			b.WriteRune(' ')
			continue
		}
		if r == '(' {
			b.WriteRune('#')
			continue
		}
		if r == '#' {
			b.WriteRune('$')
			continue
		}
		if r == '’' || r == '‘' {
			b.WriteRune('\'')
			continue
		}
		if r == 'ı' {
			b.WriteRune('j')
			continue
		}
		b.WriteRune(foldLatinAccent(r))
	}
	return b.String()
}
