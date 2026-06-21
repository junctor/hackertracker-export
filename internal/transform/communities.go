package transform

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/junctor/hackertracker-export/pkg/hackertracker"
)

type OrganizationCard struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	LogoURL string `json:"logoUrl,omitempty"`
}

type organizationCardEntry struct {
	Card   OrganizationCard
	TagIDs []int
}

type PeopleCard struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Title     string `json:"title,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

type TagSummary struct {
	ColorBackground string `json:"colorBackground"`
	ColorForeground string `json:"colorForeground"`
	ID              int    `json:"id"`
	Label           string `json:"label"`
	SortOrder       *int   `json:"sortOrder"`
}

type TagTypeBrowse struct {
	Category  *string      `json:"category"`
	ID        int          `json:"id"`
	Label     string       `json:"label"`
	SortOrder *int         `json:"sortOrder"`
	Tags      []TagSummary `json:"tags"`
}

type DocumentListItem struct {
	ID          int     `json:"id"`
	TitleText   *string `json:"titleText"`
	UpdatedAtMs int64   `json:"updatedAtMs"`
}

type CompactTag struct {
	ColorBackground string `json:"colorBackground"`
	ColorForeground string `json:"colorForeground"`
	ID              int    `json:"id"`
	Label           string `json:"label"`
}

type ContentCard struct {
	ID    int          `json:"id"`
	Tags  []CompactTag `json:"tags"`
	Title string       `json:"title"`
}

type SearchItem struct {
	ID   int    `json:"id"`
	Norm string `json:"norm"`
	Text string `json:"text"`
	Type string `json:"type"`
}

type TagIDsByLabel struct {
	ByLabel    map[string]int   `json:"byLabel"`
	Version    int              `json:"version"`
	Collisions map[string][]int `json:"collisions,omitempty"`
}

func buildViews(st *stores) map[string]any {
	return map[string]any{
		"contentCards":       buildContentCards(st),
		"documentsList":      buildDocumentsList(st),
		"organizationsCards": buildOrganizationsCards(st),
		"peopleCards":        buildPeopleCards(st),
		"searchData":         createSearchData(st),
		"tagTypesBrowse":     buildTagTypesBrowse(st),
	}
}

func buildOrganizationsCards(st *stores) map[string][]OrganizationCard {
	entries := []organizationCardEntry{}
	for _, orgID := range st.organizationIDs {
		org := st.organizationsByID[orgID]
		entries = append(entries, organizationCardEntry{
			Card: OrganizationCard{
				ID:      org.ID,
				Name:    org.Name,
				LogoURL: org.LogoURL,
			},
			TagIDs: slices.Clone(org.TagIDs),
		})
	}
	slices.SortFunc(entries, func(left, right organizationCardEntry) int {
		return cmp.Or(
			alphaCompare(left.Card.Name, right.Card.Name),
			cmp.Compare(left.Card.ID, right.Card.ID),
		)
	})

	cards := map[string][]OrganizationCard{}
	for _, entry := range entries {
		seenTags := map[int]bool{}
		assigned := false
		for _, tagID := range entry.TagIDs {
			if seenTags[tagID] {
				continue
			}
			seenTags[tagID] = true
			key := idKey(tagID)
			cards[key] = append(cards[key], entry.Card)
			assigned = true
		}
		if !assigned {
			cards["uncategorized"] = append(cards["uncategorized"], entry.Card)
		}
	}
	return cards
}

func buildPeopleCards(st *stores) []PeopleCard {
	cards := []PeopleCard{}
	for _, personID := range st.peopleIDs {
		person := st.peopleByID[personID]
		cards = append(cards, PeopleCard{
			ID:        person.ID,
			Name:      person.Name,
			Title:     person.Title,
			AvatarURL: person.AvatarURL,
		})
	}
	slices.SortFunc(cards, func(left, right PeopleCard) int {
		return cmp.Or(
			alphaCompare(left.Name, right.Name),
			cmp.Compare(left.ID, right.ID),
		)
	})
	return cards
}

func buildTagTypesBrowse(st *stores) []TagTypeBrowse {
	tagsByType := map[int][]TagSummary{}
	for _, tagID := range st.tagIDs {
		tag := st.tagsByID[tagID]
		if tag.TagTypeID == 0 {
			continue
		}
		tagsByType[tag.TagTypeID] = append(tagsByType[tag.TagTypeID], tagSummary(tag))
	}

	out := []TagTypeBrowse{}
	for _, typeID := range st.tagTypeIDs {
		tagType := st.tagTypesByID[typeID]
		if !tagType.IsBrowsable || tagType.Category == nil || *tagType.Category != "content" {
			continue
		}
		tags := tagsByType[typeID]
		if len(tags) == 0 {
			continue
		}
		slices.SortFunc(tags, compareTagSummaries)
		out = append(out, TagTypeBrowse{
			Category:  tagType.Category,
			ID:        tagType.ID,
			Label:     tagType.Label,
			SortOrder: tagType.SortOrder,
			Tags:      tags,
		})
	}
	slices.SortFunc(out, func(left, right TagTypeBrowse) int {
		return cmp.Or(
			compareOptionalInts(left.SortOrder, right.SortOrder),
			cmp.Compare(left.Label, right.Label),
			cmp.Compare(left.ID, right.ID),
		)
	})
	return out
}

func buildDocumentsList(st *stores) []DocumentListItem {
	out := []DocumentListItem{}
	for _, docID := range st.documentIDs {
		doc := st.documentsByID[docID]
		out = append(out, DocumentListItem{ID: doc.ID, TitleText: doc.TitleText, UpdatedAtMs: valueOrZero(doc.UpdatedAtMs)})
	}
	slices.SortFunc(out, func(left, right DocumentListItem) int {
		return cmp.Or(
			cmp.Compare(right.UpdatedAtMs, left.UpdatedAtMs),
			cmp.Compare(left.ID, right.ID),
		)
	})
	return out
}

func buildContentCards(st *stores) []ContentCard {
	out := []ContentCard{}
	for _, contentID := range st.contentIDs {
		item := st.contentByID[contentID]
		tags := tagsForIDs(item.TagIDs, st.tagsByID)
		slices.SortFunc(tags, compareTags)
		compactTags := make([]CompactTag, 0, len(tags))
		for _, tag := range tags {
			compactTags = append(compactTags, compactTag(tag))
		}
		out = append(out, ContentCard{ID: item.ID, Tags: compactTags, Title: item.Title})
	}
	slices.SortFunc(out, func(left, right ContentCard) int {
		return cmp.Or(
			alphaCompare(left.Title, right.Title),
			cmp.Compare(left.ID, right.ID),
		)
	})
	return out
}

func createSearchData(st *stores) []SearchItem {
	items := []SearchItem{}
	for _, personID := range st.peopleIDs {
		person := st.peopleByID[personID]
		items = append(items, SearchItem{ID: person.ID, Norm: normalizeForSearch(person.Name), Text: person.Name, Type: "person"})
	}
	for _, contentID := range st.contentIDs {
		item := st.contentByID[contentID]
		items = append(items, SearchItem{ID: item.ID, Norm: normalizeForSearch(item.Title), Text: item.Title, Type: "content"})
	}
	for _, orgID := range st.organizationIDs {
		org := st.organizationsByID[orgID]
		items = append(items, SearchItem{ID: org.ID, Norm: normalizeForSearch(org.Name), Text: org.Name, Type: "organization"})
	}
	slices.SortStableFunc(items, func(left, right SearchItem) int {
		return alphaCompare(left.Text, right.Text)
	})
	return items
}

func buildTagIDsByLabel(tagTypes []hackertracker.TagType) TagIDsByLabel {
	byLabel := map[string]int{}
	collisions := map[string]map[int]bool{}
	for _, tagType := range tagTypes {
		for _, tag := range tagType.Tags {
			id, ok := normalizeID(tag.ID)
			if !ok {
				continue
			}
			key := normalizeLabel(tag.Label)
			if key == "" {
				continue
			}
			existingID, exists := byLabel[key]
			if !exists {
				byLabel[key] = id
				continue
			}
			if existingID != id {
				if id < existingID {
					byLabel[key] = id
				}
				if collisions[key] == nil {
					collisions[key] = map[int]bool{}
				}
				collisions[key][existingID] = true
				collisions[key][id] = true
			}
		}
	}
	result := TagIDsByLabel{ByLabel: byLabel, Version: 1}
	if len(collisions) > 0 {
		result.Collisions = map[string][]int{}
		for key, set := range collisions {
			result.Collisions[key] = slices.Sorted(maps.Keys(set))
		}
	}
	return result
}

func normalizeLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func compactTag(tag TagModel) CompactTag {
	return CompactTag{
		ColorBackground: tag.ColorBackground,
		ColorForeground: tag.ColorForeground,
		ID:              tag.ID,
		Label:           tag.Label,
	}
}

func tagSummary(tag TagModel) TagSummary {
	return TagSummary{
		ColorBackground: tag.ColorBackground,
		ColorForeground: tag.ColorForeground,
		ID:              tag.ID,
		Label:           tag.Label,
		SortOrder:       tag.SortOrder,
	}
}

func compareTagSummaries(a, b TagSummary) int {
	return cmp.Or(
		compareOptionalInts(a.SortOrder, b.SortOrder),
		alphaCompare(a.Label, b.Label),
		cmp.Compare(a.ID, b.ID),
	)
}
