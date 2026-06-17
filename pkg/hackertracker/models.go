package hackertracker

type Conference struct {
	Code     string `json:"code" firestore:"code"`
	Name     string `json:"name" firestore:"name"`
	Timezone string `json:"timezone" firestore:"timezone"`
}

type Link struct {
	Label string `json:"label,omitempty" firestore:"label"`
	Type  string `json:"type,omitempty" firestore:"type"`
	URL   string `json:"url,omitempty" firestore:"url"`
}

type Asset struct {
	URL string `json:"url,omitempty" firestore:"url"`
}

type Ref struct {
	ID any `json:"id,omitempty" firestore:"id"`
}

type Article struct {
	ID           any     `json:"id" firestore:"id"`
	Name         string  `json:"name" firestore:"name"`
	Text         *string `json:"text,omitempty" firestore:"text"`
	UpdatedAt    any     `json:"updated_at,omitempty" firestore:"updated_at"`
	UpdatedTSZ   string  `json:"updated_tsz,omitempty" firestore:"updated_tsz"`
	UpdatedAtStr string  `json:"updated_at_str,omitempty" firestore:"updated_at_str"`
}

type Content struct {
	ID                any              `json:"id" firestore:"id"`
	Title             string           `json:"title" firestore:"title"`
	Description       string           `json:"description,omitempty" firestore:"description"`
	Links             []map[string]any `json:"links,omitempty" firestore:"links"`
	People            []ContentPerson  `json:"people,omitempty" firestore:"people"`
	Sessions          []Session        `json:"sessions,omitempty" firestore:"sessions"`
	TagIDs            []any            `json:"tag_ids,omitempty" firestore:"tag_ids"`
	RelatedContentIDs []any            `json:"related_content_ids,omitempty" firestore:"related_content_ids"`
}

type ContentPerson struct {
	PersonID  any `json:"person_id" firestore:"person_id"`
	SortOrder any `json:"sort_order,omitempty" firestore:"sort_order"`
}

type Session struct {
	SessionID  any    `json:"session_id" firestore:"session_id"`
	BeginTSZ   string `json:"begin_tsz,omitempty" firestore:"begin_tsz"`
	EndTSZ     string `json:"end_tsz,omitempty" firestore:"end_tsz"`
	LocationID any    `json:"location_id,omitempty" firestore:"location_id"`
}

type Document struct {
	ID           any    `json:"id" firestore:"id"`
	TitleText    string `json:"title_text,omitempty" firestore:"title_text"`
	BodyText     string `json:"body_text,omitempty" firestore:"body_text"`
	UpdatedAt    any    `json:"updated_at,omitempty" firestore:"updated_at"`
	UpdatedTSZ   string `json:"updated_tsz,omitempty" firestore:"updated_tsz"`
	UpdatedAtStr string `json:"updated_at_str,omitempty" firestore:"updated_at_str"`
}

type Event struct {
	ID         any           `json:"id" firestore:"id"`
	Title      string        `json:"title" firestore:"title"`
	BeginTSZ   string        `json:"begin_tsz,omitempty" firestore:"begin_tsz"`
	EndTSZ     string        `json:"end_tsz,omitempty" firestore:"end_tsz"`
	Location   *Ref          `json:"location,omitempty" firestore:"location"`
	LocationID any           `json:"location_id,omitempty" firestore:"location_id"`
	ContentID  any           `json:"content_id,omitempty" firestore:"content_id"`
	Speakers   []Ref         `json:"speakers,omitempty" firestore:"speakers"`
	People     []EventPerson `json:"people,omitempty" firestore:"people"`
	TagIDs     []any         `json:"tag_ids,omitempty" firestore:"tag_ids"`
	Type       *EventType    `json:"type,omitempty" firestore:"type"`
}

type EventPerson struct {
	PersonID any `json:"person_id" firestore:"person_id"`
}

type EventType struct {
	Color string `json:"color,omitempty" firestore:"color"`
}

type Location struct {
	ID        any    `json:"id" firestore:"id"`
	Name      string `json:"name" firestore:"name"`
	ShortName string `json:"short_name,omitempty" firestore:"short_name"`
	ParentID  any    `json:"parent_id,omitempty" firestore:"parent_id"`
}

type Organization struct {
	ID               any              `json:"id" firestore:"id"`
	Name             string           `json:"name" firestore:"name"`
	Description      string           `json:"description,omitempty" firestore:"description"`
	Links            []map[string]any `json:"links,omitempty" firestore:"links"`
	Logo             *Asset           `json:"logo,omitempty" firestore:"logo"`
	TagIDAsOrganizer any              `json:"tag_id_as_organizer,omitempty" firestore:"tag_id_as_organizer"`
	TagIDs           []any            `json:"tag_ids,omitempty" firestore:"tag_ids"`
}

type Speaker struct {
	ID           any              `json:"id" firestore:"id"`
	Name         string           `json:"name" firestore:"name"`
	Description  string           `json:"description,omitempty" firestore:"description"`
	Pronouns     string           `json:"pronouns,omitempty" firestore:"pronouns"`
	Title        string           `json:"title,omitempty" firestore:"title"`
	Affiliations []any            `json:"affiliations,omitempty" firestore:"affiliations"`
	Avatar       *Asset           `json:"avatar,omitempty" firestore:"avatar"`
	Links        []map[string]any `json:"links,omitempty" firestore:"links"`
	ContentIDs   []any            `json:"content_ids,omitempty" firestore:"content_ids"`
}

type TagType struct {
	ID          any    `json:"id" firestore:"id"`
	Label       string `json:"label" firestore:"label"`
	Category    string `json:"category,omitempty" firestore:"category"`
	SortOrder   any    `json:"sort_order,omitempty" firestore:"sort_order"`
	IsBrowsable bool   `json:"is_browsable,omitempty" firestore:"is_browsable"`
	Tags        []Tag  `json:"tags,omitempty" firestore:"tags"`
}

type Tag struct {
	ID              any    `json:"id" firestore:"id"`
	Label           string `json:"label" firestore:"label"`
	ColorBackground string `json:"color_background,omitempty" firestore:"color_background"`
	ColorForeground string `json:"color_foreground,omitempty" firestore:"color_foreground"`
	SortOrder       any    `json:"sort_order,omitempty" firestore:"sort_order"`
}

type SourceData struct {
	Articles      []Article
	Content       []Content
	Documents     []Document
	Events        []Event
	Locations     []Location
	Menus         []map[string]any
	Organizations []Organization
	Speakers      []Speaker
	TagTypes      []TagType
}

var Collections = []string{
	"articles",
	"content",
	"documents",
	"events",
	"locations",
	"menus",
	"organizations",
	"speakers",
	"tagtypes",
}
