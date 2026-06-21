package hackertracker

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

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
	ID json.Number `json:"id,omitempty" firestore:"id"`
}

type Article struct {
	ID           json.Number `json:"id" firestore:"id"`
	Name         string      `json:"name" firestore:"name"`
	Text         *string     `json:"text,omitempty" firestore:"text"`
	UpdatedAt    string      `json:"updated_at,omitempty" firestore:"updated_at"`
	UpdatedTSZ   string      `json:"updated_tsz,omitempty" firestore:"updated_tsz"`
	UpdatedAtStr string      `json:"updated_at_str,omitempty" firestore:"updated_at_str"`
}

type Content struct {
	ID                json.Number     `json:"id" firestore:"id"`
	Title             string          `json:"title" firestore:"title"`
	Description       string          `json:"description,omitempty" firestore:"description"`
	Links             []Link          `json:"links,omitempty" firestore:"links"`
	People            []ContentPerson `json:"people,omitempty" firestore:"people"`
	Sessions          []Session       `json:"sessions,omitempty" firestore:"sessions"`
	TagIDs            []json.Number   `json:"tag_ids,omitempty" firestore:"tag_ids"`
	RelatedContentIDs []json.Number   `json:"related_content_ids,omitempty" firestore:"related_content_ids"`
}

type ContentPerson struct {
	PersonID  json.Number `json:"person_id" firestore:"person_id"`
	SortOrder json.Number `json:"sort_order,omitempty" firestore:"sort_order"`
}

type Session struct {
	SessionID  json.Number `json:"session_id" firestore:"session_id"`
	BeginTSZ   string      `json:"begin_tsz,omitempty" firestore:"begin_tsz"`
	EndTSZ     string      `json:"end_tsz,omitempty" firestore:"end_tsz"`
	LocationID json.Number `json:"location_id,omitempty" firestore:"location_id"`
}

type Document struct {
	ID           json.Number `json:"id" firestore:"id"`
	TitleText    string      `json:"title_text,omitempty" firestore:"title_text"`
	BodyText     string      `json:"body_text,omitempty" firestore:"body_text"`
	UpdatedAt    string      `json:"updated_at,omitempty" firestore:"updated_at"`
	UpdatedTSZ   string      `json:"updated_tsz,omitempty" firestore:"updated_tsz"`
	UpdatedAtStr string      `json:"updated_at_str,omitempty" firestore:"updated_at_str"`
}

type Event struct {
	ID         json.Number   `json:"id" firestore:"id"`
	Title      string        `json:"title" firestore:"title"`
	BeginTSZ   string        `json:"begin_tsz,omitempty" firestore:"begin_tsz"`
	EndTSZ     string        `json:"end_tsz,omitempty" firestore:"end_tsz"`
	Location   *Ref          `json:"location,omitempty" firestore:"location"`
	LocationID json.Number   `json:"location_id,omitempty" firestore:"location_id"`
	ContentID  json.Number   `json:"content_id,omitempty" firestore:"content_id"`
	Speakers   []Ref         `json:"speakers,omitempty" firestore:"speakers"`
	People     []EventPerson `json:"people,omitempty" firestore:"people"`
	TagIDs     []json.Number `json:"tag_ids,omitempty" firestore:"tag_ids"`
	Type       *EventType    `json:"type,omitempty" firestore:"type"`
}

type EventPerson struct {
	PersonID json.Number `json:"person_id" firestore:"person_id"`
}

type EventType struct {
	Color string `json:"color,omitempty" firestore:"color"`
}

type Location struct {
	ID        json.Number `json:"id" firestore:"id"`
	Name      string      `json:"name" firestore:"name"`
	ShortName string      `json:"short_name,omitempty" firestore:"short_name"`
	ParentID  json.Number `json:"parent_id,omitempty" firestore:"parent_id"`
}

type Organization struct {
	ID               json.Number   `json:"id" firestore:"id"`
	Name             string        `json:"name" firestore:"name"`
	Description      string        `json:"description,omitempty" firestore:"description"`
	Links            []Link        `json:"links,omitempty" firestore:"links"`
	Logo             *Asset        `json:"logo,omitempty" firestore:"logo"`
	TagIDAsOrganizer json.Number   `json:"tag_id_as_organizer,omitempty" firestore:"tag_id_as_organizer"`
	TagIDs           []json.Number `json:"tag_ids,omitempty" firestore:"tag_ids"`
}

type Speaker struct {
	ID           json.Number   `json:"id" firestore:"id"`
	Name         string        `json:"name" firestore:"name"`
	Description  string        `json:"description,omitempty" firestore:"description"`
	Pronouns     string        `json:"pronouns,omitempty" firestore:"pronouns"`
	Title        string        `json:"title,omitempty" firestore:"title"`
	Affiliations Affiliations  `json:"affiliations,omitempty" firestore:"affiliations"`
	Avatar       *Asset        `json:"avatar,omitempty" firestore:"avatar"`
	Links        []Link        `json:"links,omitempty" firestore:"links"`
	ContentIDs   []json.Number `json:"content_ids,omitempty" firestore:"content_ids"`
}

type Affiliations []string

func (a *Affiliations) UnmarshalJSON(data []byte) error {
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err == nil {
		values := make([]string, 0, len(items))
		for i, item := range items {
			value, err := decodeAffiliation(item)
			if err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
			if value != "" {
				values = append(values, value)
			}
		}
		*a = values
		return nil
	}

	value, err := decodeAffiliation(data)
	if err != nil {
		return err
	}
	if value == "" {
		*a = nil
	} else {
		*a = []string{value}
	}
	return nil
}

func decodeAffiliation(data []byte) (string, error) {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		return strings.TrimSpace(value), nil
	}

	var item struct {
		Organization string `json:"organization"`
		Name         string `json:"name"`
		Label        string `json:"label"`
		Value        string `json:"value"`
		Title        string `json:"title"`
	}
	if err := json.Unmarshal(data, &item); err != nil {
		return "", err
	}
	for _, candidate := range []string{item.Organization, item.Name, item.Label, item.Value, item.Title} {
		if candidate = strings.TrimSpace(candidate); candidate != "" {
			return candidate, nil
		}
	}
	return "", nil
}

type TagType struct {
	ID          json.Number `json:"id" firestore:"id"`
	Label       string      `json:"label" firestore:"label"`
	Category    string      `json:"category,omitempty" firestore:"category"`
	SortOrder   json.Number `json:"sort_order,omitempty" firestore:"sort_order"`
	IsBrowsable bool        `json:"is_browsable,omitempty" firestore:"is_browsable"`
	Tags        []Tag       `json:"tags,omitempty" firestore:"tags"`
}

type Tag struct {
	ID              json.Number `json:"id" firestore:"id"`
	Label           string      `json:"label" firestore:"label"`
	ColorBackground string      `json:"color_background,omitempty" firestore:"color_background"`
	ColorForeground string      `json:"color_foreground,omitempty" firestore:"color_foreground"`
	SortOrder       json.Number `json:"sort_order,omitempty" firestore:"sort_order"`
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

var collections = [...]string{
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

func CollectionNames() []string {
	return slices.Clone(collections[:])
}
