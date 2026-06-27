package transform

import (
	"cmp"
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
	contentByID       map[int]ContentModel
	sessionsByID      map[int]SessionModel
	peopleByID        map[int]PersonModel
	locationsByID     map[int]LocationModel
	organizationsByID map[int]OrganizationModel
	tagsByID          map[int]TagModel
	tagTypesByID      map[int]TagTypeModel
	documentsByID     map[int]DocumentModel
	articlesByID      map[int]ArticleModel

	contentIDs      []int
	sessionIDs      []int
	peopleIDs       []int
	locationIDs     []int
	organizationIDs []int
	tagIDs          []int
	tagTypeIDs      []int
	documentIDs     []int
	articleIDs      []int
}

type EntityStore[T any] struct {
	AllIDs []int        `json:"allIds"`
	ByID   map[string]T `json:"byId"`
}

type LinkModel struct {
	Label string `json:"label,omitempty"`
	Type  string `json:"type,omitempty"`
	URL   string `json:"url,omitempty"`
}

type ContentPersonModel struct {
	PersonID  int  `json:"personId"`
	SortOrder *int `json:"sortOrder"`
}

type ContentModel struct {
	ID                int                  `json:"id"`
	RelatedContentIDs []int                `json:"relatedContentIds"`
	Sessions          []int                `json:"sessions"`
	Title             string               `json:"title"`
	Description       string               `json:"description,omitempty"`
	Links             []LinkModel          `json:"links,omitempty"`
	TagIDs            []int                `json:"tagIds,omitempty"`
	People            []ContentPersonModel `json:"people,omitempty"`
}

type SessionModel struct {
	ID                    int    `json:"id"`
	ContentID             int    `json:"contentId"`
	Title                 string `json:"title"`
	Begin                 string `json:"begin"`
	BeginDisplay          string `json:"beginDisplay,omitempty"`
	BeginIso              string `json:"beginIso,omitempty"`
	BeginTimestampSeconds int64  `json:"beginTimestampSeconds"`
	End                   string `json:"end"`
	EndDisplay            string `json:"endDisplay,omitempty"`
	EndIso                string `json:"endIso,omitempty"`
	EndTimestampSeconds   int64  `json:"endTimestampSeconds"`
	LocationID            *int   `json:"locationId"`
	PersonIDs             []int  `json:"personIds,omitempty"`
	TagIDs                []int  `json:"tagIds,omitempty"`
	ChannelID             *int   `json:"channelId,omitempty"`
	RecordingPolicyID     int    `json:"recordingPolicyId,omitempty"`
	TimezoneName          string `json:"timezoneName,omitempty"`
}

type PersonModel struct {
	ContentIDs   []int       `json:"contentIds"`
	ID           int         `json:"id"`
	Name         string      `json:"name"`
	Description  string      `json:"description,omitempty"`
	Pronouns     string      `json:"pronouns,omitempty"`
	Title        string      `json:"title,omitempty"`
	Affiliations []string    `json:"affiliations,omitempty"`
	AvatarURL    string      `json:"avatarUrl,omitempty"`
	Links        []LinkModel `json:"links,omitempty"`
}

type LocationModel struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	ShortName *string `json:"shortName"`
	ParentID  *int    `json:"parentId"`
}

type OrganizationModel struct {
	Description      string      `json:"description"`
	ID               int         `json:"id"`
	Links            []LinkModel `json:"links"`
	Name             string      `json:"name"`
	TagIDAsOrganizer *int        `json:"tagIdAsOrganizer"`
	LogoURL          string      `json:"logoUrl,omitempty"`
	TagIDs           []int       `json:"tagIds,omitempty"`
}

type TagModel struct {
	ColorBackground string `json:"colorBackground"`
	ColorForeground string `json:"colorForeground"`
	ID              int    `json:"id"`
	Label           string `json:"label"`
	SortOrder       *int   `json:"sortOrder"`
	TagTypeID       int    `json:"tagTypeId"`
}

type TagTypeModel struct {
	Category    *string `json:"category"`
	ID          int     `json:"id"`
	IsBrowsable bool    `json:"isBrowsable"`
	Label       string  `json:"label"`
	SortOrder   *int    `json:"sortOrder"`
}

type ArticleModel struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Text        string `json:"text"`
	UpdatedAtMs *int64 `json:"updatedAtMs"`
}

type DocumentModel struct {
	BodyText    *string `json:"bodyText"`
	ID          int     `json:"id"`
	TitleText   *string `json:"titleText"`
	UpdatedAtMs *int64  `json:"updatedAtMs,omitempty"`
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
		Entities: entities(st),
		Indexes: map[string]any{
			"sessionsByDay": indexes.sessionsByDay,
			"sessionsByTag": indexes.sessionsByTag,
		},
		Views:   views,
		Derived: map[string]any{"tagIdsByLabel": buildTagIDsByLabel(data)},
		Details: details,
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
	for _, speaker := range data.Speakers {
		if id, ok := normalizeID(speaker.ID); ok {
			refs.personIDs[id] = true
		}
	}
	for _, tag := range sourceTags(data) {
		if id, ok := normalizeID(tag.ID); ok {
			refs.tagIDs[id] = true
		}
	}
	for _, item := range data.Content {
		if id, ok := normalizeID(item.ID); ok {
			refs.contentIDs[id] = true
		}
	}

	st := &stores{
		contentByID:       map[int]ContentModel{},
		sessionsByID:      map[int]SessionModel{},
		peopleByID:        map[int]PersonModel{},
		locationsByID:     map[int]LocationModel{},
		organizationsByID: map[int]OrganizationModel{},
		tagsByID:          map[int]TagModel{},
		tagTypesByID:      map[int]TagTypeModel{},
		documentsByID:     map[int]DocumentModel{},
		articlesByID:      map[int]ArticleModel{},
	}

	for _, item := range data.Content {
		content, sessions, err := buildContentModel(item, refs.personIDs, refs.tagIDs, refs.contentIDs, refs.locationIDs, timezone)
		if err != nil {
			return nil, err
		}
		putEntity(st.contentByID, &st.contentIDs, content)
		for _, session := range sessions {
			putEntity(st.sessionsByID, &st.sessionIDs, session)
		}
	}
	for _, speaker := range data.Speakers {
		model, err := buildPersonModel(speaker, refs.contentIDs)
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
		var shortName *string
		if loc.ShortName != "" {
			shortName = &loc.ShortName
		}
		var parentID *int
		if id, ok := normalizeID(loc.ParentID); ok {
			parentID = &id
		}
		putEntity(st.locationsByID, &st.locationIDs, LocationModel{ID: id, Name: loc.Name, ShortName: shortName, ParentID: parentID})
	}
	for _, org := range data.Organizations {
		model, err := buildOrganizationModel(org, refs.tagIDs)
		if err != nil {
			return nil, err
		}
		putEntity(st.organizationsByID, &st.organizationIDs, model)
	}
	for _, tag := range buildTags(data) {
		putEntity(st.tagsByID, &st.tagIDs, tag)
	}
	for _, tagType := range data.TagTypes {
		id, ok := normalizeID(tagType.ID)
		if !ok {
			return nil, fmt.Errorf("tag type missing id")
		}
		putEntity(st.tagTypesByID, &st.tagTypeIDs, TagTypeModel{
			Category:    stringPtrOrNil(tagType.Category),
			ID:          id,
			IsBrowsable: tagType.IsBrowsable,
			Label:       tagType.Label,
			SortOrder:   intPtrFromValue(tagType.SortOrder),
		})
	}
	for _, article := range data.Articles {
		id, ok := normalizeID(article.ID)
		if !ok {
			return nil, fmt.Errorf("article missing id")
		}
		putEntity(st.articlesByID, &st.articleIDs, ArticleModel{
			ID:          id,
			Name:        article.Name,
			Text:        article.Text,
			UpdatedAtMs: resolveUpdatedAtMs(article.UpdatedAt, article.UpdatedTSZ, article.UpdatedAtStr),
		})
	}
	for _, doc := range data.Documents {
		id, ok := normalizeID(doc.ID)
		if !ok {
			return nil, fmt.Errorf("document missing id")
		}
		putEntity(st.documentsByID, &st.documentIDs, DocumentModel{
			BodyText:    stringPtrOrNil(doc.BodyText),
			ID:          id,
			TitleText:   stringPtrOrNil(doc.TitleText),
			UpdatedAtMs: resolveUpdatedAtMs(doc.UpdatedAt, doc.UpdatedTSZ, doc.UpdatedAtStr),
		})
	}

	return st, nil
}

func entities(st *stores) map[string]any {
	return map[string]any{
		"content":       entityStore(st.contentIDs, st.contentByID),
		"sessions":      entityStore(st.sessionIDs, st.sessionsByID),
		"people":        entityStore(st.peopleIDs, st.peopleByID),
		"locations":     entityStore(st.locationIDs, st.locationsByID),
		"organizations": entityStore(st.organizationIDs, st.organizationsByID),
		"tags":          entityStore(st.tagIDs, st.tagsByID),
		"tagTypes":      entityStore(st.tagTypeIDs, st.tagTypesByID),
		"articles":      entityStore(st.articleIDs, st.articlesByID),
		"documents":     entityStore(st.documentIDs, st.documentsByID),
	}
}

func buildContentModel(item hackertracker.Content, personIDs, tagIDs, contentIDs, locationIDs map[int]bool, timezone string) (ContentModel, []SessionModel, error) {
	id, ok := normalizeID(item.ID)
	if !ok {
		return ContentModel{}, nil, fmt.Errorf("content item missing id")
	}

	tagIDsOut := uniqueIDs(item.TagIDs, tagIDs)
	slices.Sort(tagIDsOut)
	related := uniqueIDs(item.RelatedContentIDs, contentIDs)
	slices.Sort(related)

	people := buildContentPeople(item.People, personIDs)
	sessionPersonIDs := contentPersonIDs(people)

	sessions := make([]SessionModel, 0, len(item.Sessions))
	sessionIDs := make([]int, 0, len(item.Sessions))
	seenSessionIDs := map[int]bool{}
	for _, sourceSession := range item.Sessions {
		session, err := buildSessionModel(sourceSession, id, item.Title, sessionPersonIDs, tagIDsOut, locationIDs, timezone)
		if err != nil {
			return ContentModel{}, nil, fmt.Errorf("content %d: %w", id, err)
		}
		if seenSessionIDs[session.ID] {
			continue
		}
		seenSessionIDs[session.ID] = true
		sessionIDs = append(sessionIDs, session.ID)
		sessions = append(sessions, session)
	}
	slices.Sort(sessionIDs)

	return ContentModel{
		ID:                id,
		RelatedContentIDs: related,
		Sessions:          sessionIDs,
		Title:             item.Title,
		Description:       item.Description,
		Links:             linksToModels(item.Links),
		TagIDs:            tagIDsOut,
		People:            people,
	}, sessions, nil
}

func buildContentPeople(sourcePeople []hackertracker.ContentPerson, validPersonIDs map[int]bool) []ContentPersonModel {
	peopleByID := map[int]ContentPersonModel{}
	for _, person := range sourcePeople {
		personID, ok := normalizeID(person.PersonID)
		if !ok || !validPersonIDs[personID] {
			continue
		}
		if _, exists := peopleByID[personID]; exists {
			continue
		}
		peopleByID[personID] = ContentPersonModel{PersonID: personID, SortOrder: intPtrFromValue(person.SortOrder)}
	}

	people := make([]ContentPersonModel, 0, len(peopleByID))
	for _, person := range peopleByID {
		people = append(people, person)
	}
	slices.SortFunc(people, compareContentPeople)
	return people
}

func buildSessionModel(sourceSession hackertracker.Session, contentID int, contentTitle string, personIDs []int, tagIDs []int, validLocationIDs map[int]bool, timezone string) (SessionModel, error) {
	id, ok := sourceSessionID(sourceSession)
	if !ok {
		return SessionModel{}, fmt.Errorf("session missing id")
	}

	var locationID *int
	if id, ok := normalizeID(sourceSession.LocationID); ok && (validLocationIDs == nil || validLocationIDs[id]) {
		locationID = &id
	}

	begin := timestampString(sourceSession.BeginTSZ, sourceSession.BeginTimestamp)
	end := timestampString(sourceSession.EndTSZ, sourceSession.EndTimestamp)

	return SessionModel{
		ID:                    id,
		ContentID:             contentID,
		Title:                 contentTitle,
		Begin:                 begin,
		BeginDisplay:          sessionTimeTable(begin, true, timezone),
		BeginIso:              isoTime(begin),
		BeginTimestampSeconds: timestampSeconds(begin),
		End:                   end,
		EndDisplay:            sessionTimeTable(end, false, timezone),
		EndIso:                isoTime(end),
		EndTimestampSeconds:   timestampSeconds(end),
		LocationID:            locationID,
		PersonIDs:             slices.Clone(personIDs),
		TagIDs:                slices.Clone(tagIDs),
		ChannelID:             sourceSession.ChannelID,
		RecordingPolicyID:     sourceSession.RecordingPolicyID,
		TimezoneName:          sourceSession.TimezoneName,
	}, nil
}

func buildPersonModel(person hackertracker.Speaker, contentIDs map[int]bool) (PersonModel, error) {
	id, ok := normalizeID(person.ID)
	if !ok {
		return PersonModel{}, fmt.Errorf("person missing id")
	}
	rawContentIDs := person.ContentIDs
	if len(rawContentIDs) == 0 {
		rawContentIDs = person.LegacyContentIDs
	}
	contentIDsOut := uniqueIDs(rawContentIDs, contentIDs)
	slices.Sort(contentIDsOut)

	return PersonModel{
		ContentIDs:   contentIDsOut,
		ID:           id,
		Name:         person.Name,
		Description:  person.Description,
		Pronouns:     person.Pronouns,
		Title:        person.Title,
		Affiliations: slices.Clone([]string(person.Affiliations)),
		AvatarURL:    avatarURL(person.Avatar),
		Links:        linksToModels(person.Links),
	}, nil
}

func buildOrganizationModel(org hackertracker.Organization, tagIDs map[int]bool) (OrganizationModel, error) {
	id, ok := normalizeID(org.ID)
	if !ok {
		return OrganizationModel{}, fmt.Errorf("organization missing id")
	}
	tags := uniqueIDs(org.TagIDs, tagIDs)
	slices.Sort(tags)

	var tagIDAsOrganizer *int
	if org.TagIDAsOrganizer != nil {
		if tagID, ok := normalizeID(*org.TagIDAsOrganizer); ok {
			tagIDAsOrganizer = &tagID
		}
	}

	return OrganizationModel{
		Description:      org.Description,
		ID:               id,
		Links:            linksToModels(org.Links),
		Name:             org.Name,
		TagIDAsOrganizer: tagIDAsOrganizer,
		LogoURL:          org.Logo.URL,
		TagIDs:           tags,
	}, nil
}

func buildTags(data hackertracker.SourceData) []TagModel {
	tags := []TagModel{}
	for _, tag := range sourceTags(data) {
		id, ok := normalizeID(tag.ID)
		if !ok {
			continue
		}
		tags = append(tags, TagModel{
			ColorBackground: tag.ColorBackground,
			ColorForeground: tag.ColorForeground,
			ID:              id,
			Label:           tag.Label,
			SortOrder:       intPtrFromValue(tag.SortOrder),
			TagTypeID:       tag.TagTypeID,
		})
	}
	slices.SortFunc(tags, compareTags)
	return tags
}

func sourceTags(data hackertracker.SourceData) []hackertracker.Tag {
	tags := []hackertracker.Tag{}
	for _, group := range data.TagTypes {
		typeID, _ := normalizeID(group.ID)
		for _, tag := range group.Tags {
			tag.TagTypeID = typeID
			tags = append(tags, tag)
		}
	}
	return tags
}

func sourceSessionID(session hackertracker.Session) (int, bool) {
	return normalizeID(session.SessionID)
}

func contentPersonIDs(people []ContentPersonModel) []int {
	ids := make([]int, 0, len(people))
	for _, person := range people {
		ids = append(ids, person.PersonID)
	}
	return ids
}

func putEntity[T interface{ entityID() int }](store map[int]T, ids *[]int, model T) {
	id := model.entityID()
	if _, exists := store[id]; exists {
		return
	}
	store[id] = model
	*ids = append(*ids, id)
}

func (m ContentModel) entityID() int      { return m.ID }
func (m SessionModel) entityID() int      { return m.ID }
func (m PersonModel) entityID() int       { return m.ID }
func (m LocationModel) entityID() int     { return m.ID }
func (m OrganizationModel) entityID() int { return m.ID }
func (m TagModel) entityID() int          { return m.ID }
func (m TagTypeModel) entityID() int      { return m.ID }
func (m ArticleModel) entityID() int      { return m.ID }
func (m DocumentModel) entityID() int     { return m.ID }

func entityStore[T any](ids []int, byID map[int]T) EntityStore[T] {
	slices.Sort(ids)
	byIDOut := map[string]T{}
	for _, id := range ids {
		byIDOut[fmt.Sprint(id)] = byID[id]
	}
	return EntityStore[T]{AllIDs: slices.Clone(ids), ByID: byIDOut}
}

func linksToModels(links []hackertracker.Link) []LinkModel {
	out := make([]LinkModel, 0, len(links))
	for _, link := range links {
		if link.Label == "" && link.Type == "" && link.URL == "" {
			continue
		}
		out = append(out, LinkModel{Label: link.Label, Type: link.Type, URL: link.URL})
	}
	return out
}

func compareTags(a, b TagModel) int {
	return cmp.Or(
		compareOptionalInts(a.SortOrder, b.SortOrder),
		alphaCompare(a.Label, b.Label),
		cmp.Compare(a.ID, b.ID),
	)
}

func compareContentPeople(a, b ContentPersonModel) int {
	return cmp.Or(
		compareOptionalInts(a.SortOrder, b.SortOrder),
		cmp.Compare(a.PersonID, b.PersonID),
	)
}

func compareOptionalInts(a, b *int) int {
	if (a == nil) != (b == nil) {
		if a != nil {
			return -1
		}
		return 1
	}
	if a == nil {
		return 0
	}
	return cmp.Compare(*a, *b)
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

func normalizeForSearch(text string) string {
	lowered := strings.ToLower(text)
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

func avatarURL(asset *hackertracker.Asset) string {
	if asset == nil {
		return ""
	}
	return asset.URL
}
