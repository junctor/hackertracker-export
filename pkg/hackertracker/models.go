package hackertracker

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

type Conference struct {
	ID                            int              `json:"id" firestore:"id"`
	ConferenceID                  int              `json:"conference_id" firestore:"conference_id"`
	Code                          string           `json:"code" firestore:"code"`
	Name                          string           `json:"name" firestore:"name"`
	Description                   string           `json:"description" firestore:"description"`
	TaglineText                   string           `json:"tagline_text" firestore:"tagline_text"`
	Timezone                      string           `json:"timezone" firestore:"timezone"`
	BeginTSZ                      string           `json:"begin_tsz" firestore:"begin_tsz"`
	StartDate                     string           `json:"start_date" firestore:"start_date"`
	StartTimestamp                time.Time        `json:"start_timestamp" firestore:"start_timestamp"`
	StartTimestampStr             string           `json:"start_timestamp_str" firestore:"start_timestamp_str"`
	EndDate                       string           `json:"end_date" firestore:"end_date"`
	EndTimestamp                  time.Time        `json:"end_timestamp" firestore:"end_timestamp"`
	EndTimestampStr               string           `json:"end_timestamp_str" firestore:"end_timestamp_str"`
	EndTSZ                        string           `json:"end_tsz" firestore:"end_tsz"`
	KickoffTimestamp              time.Time        `json:"kickoff_timestamp" firestore:"kickoff_timestamp"`
	KickoffTimestampStr           string           `json:"kickoff_timestamp_str" firestore:"kickoff_timestamp_str"`
	KickoffTSZ                    string           `json:"kickoff_tsz" firestore:"kickoff_tsz"`
	UpdatedAt                     time.Time        `json:"updated_at" firestore:"updated_at"`
	CodeOfConduct                 string           `json:"codeofconduct" firestore:"codeofconduct"`
	EmergencyDocumentID           *int             `json:"emergency_document_id" firestore:"emergency_document_id"`
	EnableMerch                   bool             `json:"enable_merch" firestore:"enable_merch"`
	EnableMerchCart               bool             `json:"enable_merch_cart" firestore:"enable_merch_cart"`
	FeedbackFormRateLimitSeconds  int              `json:"feedbackform_ratelimit_seconds" firestore:"feedbackform_ratelimit_seconds"`
	Hidden                        bool             `json:"hidden" firestore:"hidden"`
	HomeMenuID                    int              `json:"home_menu_id" firestore:"home_menu_id"`
	Link                          string           `json:"link" firestore:"link"`
	Maps                          []map[string]any `json:"maps" firestore:"maps"`
	MerchMandatoryAcknowledgement string           `json:"merch_mandatory_acknowledgement" firestore:"merch_mandatory_acknowledgement"`
	MerchTaxStatement             string           `json:"merch_tax_statement" firestore:"merch_tax_statement"`
	SupportDoc                    string           `json:"supportdoc" firestore:"supportdoc"`
}

type Link struct {
	Label string `json:"label" firestore:"label"`
	Type  string `json:"type" firestore:"type"`
	URL   string `json:"url" firestore:"url"`
}

type Asset struct {
	AssetID    int    `json:"asset_id,omitempty" firestore:"asset_id"`
	AssetUUID  string `json:"asset_uuid,omitempty" firestore:"asset_uuid"`
	FileSize   int    `json:"filesize,omitempty" firestore:"filesize"`
	FileType   string `json:"filetype,omitempty" firestore:"filetype"`
	HashCRC32C string `json:"hash_crc32c,omitempty" firestore:"hash_crc32c"`
	HashMD5    string `json:"hash_md5,omitempty" firestore:"hash_md5"`
	HashSHA256 string `json:"hash_sha256,omitempty" firestore:"hash_sha256"`
	IsLogo     string `json:"is_logo,omitempty" firestore:"is_logo"`
	Name       string `json:"name,omitempty" firestore:"name"`
	OrgaID     int    `json:"orga_id,omitempty" firestore:"orga_id"`
	SortOrder  int    `json:"sort_order,omitempty" firestore:"sort_order"`
	URL        string `json:"url,omitempty" firestore:"url"`
}

type Ref struct {
	ID int `json:"id,omitempty" firestore:"id"`
}

type Article struct {
	ID           int       `json:"id" firestore:"id"`
	Conference   string    `json:"conference" firestore:"conference"`
	ConferenceID int       `json:"conference_id" firestore:"conference_id"`
	Name         string    `json:"name" firestore:"name"`
	Text         string    `json:"text" firestore:"text"`
	UpdatedAt    time.Time `json:"updated_at" firestore:"updated_at"`
	UpdatedTSZ   string    `json:"updated_tsz" firestore:"updated_tsz"`
	UpdatedAtStr string    `json:"updated_at_str,omitempty" firestore:"updated_at_str"`
}

type Content struct {
	ID                       int             `json:"id" firestore:"id"`
	Title                    string          `json:"title" firestore:"title"`
	Description              string          `json:"description" firestore:"description"`
	Links                    []Link          `json:"links" firestore:"links"`
	Logo                     Asset           `json:"logo" firestore:"logo"`
	Media                    []Asset         `json:"media" firestore:"media"`
	People                   []ContentPerson `json:"people" firestore:"people"`
	Sessions                 []Session       `json:"sessions" firestore:"sessions"`
	TagIDs                   []int           `json:"tag_ids" firestore:"tag_ids"`
	RelatedContentIDs        []int           `json:"related_content_ids" firestore:"related_content_ids"`
	FeedbackDisableTimestamp time.Time       `json:"feedback_disable_timestamp" firestore:"feedback_disable_timestamp"`
	FeedbackDisableTSZ       *string         `json:"feedback_disable_tsz" firestore:"feedback_disable_tsz"`
	FeedbackEnableTimestamp  time.Time       `json:"feedback_enable_timestamp" firestore:"feedback_enable_timestamp"`
	FeedbackEnableTSZ        *string         `json:"feedback_enable_tsz" firestore:"feedback_enable_tsz"`
	FeedbackFormID           *int            `json:"feedback_form_id" firestore:"feedback_form_id"`
	UpdatedTimestamp         time.Time       `json:"updated_timestamp" firestore:"updated_timestamp"`
	UpdatedTSZ               string          `json:"updated_tsz" firestore:"updated_tsz"`
	VisibleAgeMin            *int            `json:"visible_age_min" firestore:"visible_age_min"`
}

type ContentPerson struct {
	PersonID  int   `json:"person_id" firestore:"person_id"`
	SortOrder int   `json:"sort_order" firestore:"sort_order"`
	TagIDs    []int `json:"tag_ids" firestore:"tag_ids"`
}

type Session struct {
	SessionID         int       `json:"session_id" firestore:"session_id"`
	BeginTimestamp    time.Time `json:"begin_timestamp" firestore:"begin_timestamp"`
	BeginTSZ          string    `json:"begin_tsz" firestore:"begin_tsz"`
	EndTimestamp      time.Time `json:"end_timestamp" firestore:"end_timestamp"`
	EndTSZ            string    `json:"end_tsz" firestore:"end_tsz"`
	LocationID        int       `json:"location_id" firestore:"location_id"`
	ChannelID         *int      `json:"channel_id" firestore:"channel_id"`
	RecordingPolicyID int       `json:"recordingpolicy_id" firestore:"recordingpolicy_id"`
	TimezoneName      string    `json:"timezone_name" firestore:"timezone_name"`
}

type Document struct {
	ID           int       `json:"id" firestore:"id"`
	Conference   string    `json:"conference" firestore:"conference"`
	ConferenceID int       `json:"conference_id" firestore:"conference_id"`
	TitleText    string    `json:"title_text" firestore:"title_text"`
	BodyText     string    `json:"body_text" firestore:"body_text"`
	UpdatedAt    time.Time `json:"updated_at" firestore:"updated_at"`
	UpdatedTSZ   string    `json:"updated_tsz" firestore:"updated_tsz"`
	UpdatedAtStr string    `json:"updated_at_str,omitempty" firestore:"updated_at_str"`
}

type Location struct {
	ID              int              `json:"id" firestore:"id"`
	Name            string           `json:"name" firestore:"name"`
	ShortName       string           `json:"short_name" firestore:"short_name"`
	ParentID        int              `json:"parent_id" firestore:"parent_id"`
	DefaultStatus   string           `json:"default_status" firestore:"default_status"`
	HierDepth       int              `json:"hier_depth" firestore:"hier_depth"`
	HierExtentLeft  int              `json:"hier_extent_left" firestore:"hier_extent_left"`
	HierExtentRight int              `json:"hier_extent_right" firestore:"hier_extent_right"`
	Hotel           string           `json:"hotel" firestore:"hotel"`
	PeerSortOrder   int              `json:"peer_sort_order" firestore:"peer_sort_order"`
	Schedule        []map[string]any `json:"schedule" firestore:"schedule"`
}

type Menu struct {
	ID           int        `json:"id" firestore:"id"`
	Conference   string     `json:"conference" firestore:"conference"`
	ConferenceID int        `json:"conference_id" firestore:"conference_id"`
	TitleText    string     `json:"title_text" firestore:"title_text"`
	Items        []MenuItem `json:"items" firestore:"items"`
}

type MenuItem struct {
	ID                   int    `json:"id" firestore:"id"`
	TitleText            string `json:"title_text" firestore:"title_text"`
	Function             string `json:"function" firestore:"function"`
	SortOrder            int    `json:"sort_order" firestore:"sort_order"`
	AppleSFSymbol        string `json:"apple_sfsymbol" firestore:"apple_sfsymbol"`
	GoogleMaterialSymbol string `json:"google_materialsymbol" firestore:"google_materialsymbol"`
	AppliedTagIDs        []int  `json:"applied_tag_ids" firestore:"applied_tag_ids"`
	DocumentID           *int   `json:"document_id" firestore:"document_id"`
	MenuID               *int   `json:"menu_id" firestore:"menu_id"`
	ProhibitTagFilter    string `json:"prohibit_tag_filter" firestore:"prohibit_tag_filter"`
}

type Organization struct {
	ID               int              `json:"id" firestore:"id"`
	Conference       string           `json:"conference" firestore:"conference"`
	ConferenceID     int              `json:"conference_id" firestore:"conference_id"`
	Name             string           `json:"name" firestore:"name"`
	Description      string           `json:"description" firestore:"description"`
	Documents        []map[string]any `json:"documents" firestore:"documents"`
	Links            []Link           `json:"links" firestore:"links"`
	Locations        []map[string]any `json:"locations" firestore:"locations"`
	Logo             Asset            `json:"logo" firestore:"logo"`
	Media            []Asset          `json:"media" firestore:"media"`
	People           []map[string]any `json:"people" firestore:"people"`
	TagIDAsOrganizer *int             `json:"tag_id_as_organizer" firestore:"tag_id_as_organizer"`
	TagIDs           []int            `json:"tag_ids" firestore:"tag_ids"`
	UpdatedAt        string           `json:"updated_at" firestore:"updated_at"`
	UpdatedTSZ       string           `json:"updated_tsz" firestore:"updated_tsz"`
}

type Speaker struct {
	ID               int          `json:"id" firestore:"id"`
	Conference       string       `json:"conference" firestore:"conference"`
	ConferenceID     int          `json:"conference_id" firestore:"conference_id"`
	Name             string       `json:"name" firestore:"name"`
	Description      string       `json:"description" firestore:"description"`
	Pronouns         string       `json:"pronouns" firestore:"pronouns"`
	Title            string       `json:"title" firestore:"title"`
	Affiliations     Affiliations `json:"affiliations" firestore:"affiliations"`
	Avatar           *Asset       `json:"avatar" firestore:"avatar"`
	Link             string       `json:"link" firestore:"link"`
	Links            []Link       `json:"links" firestore:"links"`
	Media            []Asset      `json:"media" firestore:"media"`
	Twitter          string       `json:"twitter" firestore:"twitter"`
	ContentIDs       []int        `json:"content_ids" firestore:"content_ids"`
	EventIDs         []int        `json:"event_ids" firestore:"event_ids"`
	UpdatedAt        string       `json:"updated_at" firestore:"updated_at"`
	UpdatedTimestamp time.Time    `json:"updated_timestamp" firestore:"updated_timestamp"`
	UpdatedTSZ       string       `json:"updated_tsz" firestore:"updated_tsz"`
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
	ID             int    `json:"id" firestore:"id"`
	Conference     string `json:"conference" firestore:"conference"`
	ConferenceID   int    `json:"conference_id" firestore:"conference_id"`
	Label          string `json:"label" firestore:"label"`
	Category       string `json:"category" firestore:"category"`
	SortOrder      int    `json:"sort_order" firestore:"sort_order"`
	IsBrowsable    bool   `json:"is_browsable" firestore:"is_browsable"`
	IsSingleValued bool   `json:"is_single_valued" firestore:"is_single_valued"`
	UUID           string `json:"uuid" firestore:"uuid"`
	WellKnownUUID  string `json:"well_known_uuid" firestore:"well_known_uuid"`
	Tags           []Tag  `json:"tags" firestore:"tags"`
}

type Tag struct {
	ID              int    `json:"id" firestore:"id"`
	Label           string `json:"label" firestore:"label"`
	Description     string `json:"description" firestore:"description"`
	ColorBackground string `json:"color_background" firestore:"color_background"`
	ColorForeground string `json:"color_foreground" firestore:"color_foreground"`
	SortOrder       int    `json:"sort_order" firestore:"sort_order"`
}

type SourceData struct {
	Articles      []Article
	Conference    *Conference
	Content       []Content
	Documents     []Document
	Locations     []Location
	Menus         []Menu
	Organizations []Organization
	Speakers      []Speaker
	TagTypes      []TagType
}

var collections = [...]string{
	"articles",
	"content",
	"documents",
	"locations",
	"menus",
	"organizations",
	"speakers",
	"tagtypes",
}

func CollectionNames() []string {
	return slices.Clone(collections[:])
}
