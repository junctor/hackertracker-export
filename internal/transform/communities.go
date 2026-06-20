package transform

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/junctor/hackertracker-export/pkg/hackertracker"
)

func buildViews(st *stores) map[string]any {
	organizationsCardsList := []map[string]any{}
	for _, orgID := range st.organizationIDs {
		org := st.organizationsByID[orgID]
		card := map[string]any{"id": org["id"], "name": org["name"]}
		if org["logoUrl"] != nil {
			card["logoUrl"] = org["logoUrl"]
		}
		organizationsCardsList = append(organizationsCardsList, map[string]any{"card": card, "tagIds": intSlice(org["tagIds"])})
	}
	slices.SortFunc(organizationsCardsList, func(left, right map[string]any) int {
		a := left["card"].(map[string]any)
		b := right["card"].(map[string]any)
		if stringValue(a["name"]) != stringValue(b["name"]) {
			return alphaCompare(stringValue(a["name"]), stringValue(b["name"]))
		}
		return cmp.Compare(intValue(a["id"]), intValue(b["id"]))
	})
	organizationsCards := map[string]any{}
	for _, entry := range organizationsCardsList {
		card := entry["card"]
		seenTags := map[int]bool{}
		assigned := false
		for _, tagID := range entry["tagIds"].([]int) {
			if seenTags[tagID] {
				continue
			}
			seenTags[tagID] = true
			key := stringValue(tagID)
			list, _ := organizationsCards[key].([]any)
			list = append(list, card)
			organizationsCards[key] = list
			assigned = true
		}
		if !assigned {
			list, _ := organizationsCards["uncategorized"].([]any)
			list = append(list, card)
			organizationsCards["uncategorized"] = list
		}
	}

	peopleCards := []any{}
	for _, personID := range st.peopleIDs {
		person := st.peopleByID[personID]
		model := map[string]any{"id": person["id"], "name": person["name"]}
		if person["title"] != nil {
			model["title"] = person["title"]
		}
		if person["avatarUrl"] != nil {
			model["avatarUrl"] = person["avatarUrl"]
		}
		peopleCards = append(peopleCards, model)
	}
	slices.SortFunc(peopleCards, func(left, right any) int {
		a := left.(map[string]any)
		b := right.(map[string]any)
		if stringValue(a["name"]) != stringValue(b["name"]) {
			return alphaCompare(stringValue(a["name"]), stringValue(b["name"]))
		}
		return cmp.Compare(intValue(a["id"]), intValue(b["id"]))
	})

	tagTypesBrowse := buildTagTypesBrowse(st)
	documentsList := buildDocumentsList(st)
	contentCards := buildContentCards(st)
	searchData := createSearchData(st)
	return map[string]any{
		"contentCards":       contentCards,
		"documentsList":      documentsList,
		"organizationsCards": organizationsCards,
		"peopleCards":        peopleCards,
		"searchData":         searchData,
		"tagTypesBrowse":     tagTypesBrowse,
	}
}

func buildTagTypesBrowse(st *stores) []any {
	tagsByType := map[int][]map[string]any{}
	for _, tagID := range st.tagIDs {
		tag := st.tagsByID[tagID]
		typeID := intValue(tag["tagTypeId"])
		if typeID == 0 {
			continue
		}
		tagsByType[typeID] = append(tagsByType[typeID], map[string]any{
			"colorBackground": tag["colorBackground"],
			"colorForeground": tag["colorForeground"],
			"id":              tag["id"],
			"label":           tag["label"],
			"sortOrder":       tag["sortOrder"],
		})
	}
	out := []any{}
	for _, typeID := range st.tagTypeIDs {
		tagType := st.tagTypesByID[typeID]
		if tagType["isBrowsable"] != true || tagType["category"] != "content" {
			continue
		}
		tags := tagsByType[typeID]
		if len(tags) == 0 {
			continue
		}
		slices.SortFunc(tags, compareTags)
		out = append(out, map[string]any{
			"category":  tagType["category"],
			"id":        tagType["id"],
			"label":     tagType["label"],
			"sortOrder": tagType["sortOrder"],
			"tags":      eventsAny(tags),
		})
	}
	slices.SortFunc(out, func(left, right any) int {
		a := left.(map[string]any)
		b := right.(map[string]any)
		ao := intValue(a["sortOrder"])
		bo := intValue(b["sortOrder"])
		if ao != bo {
			return cmp.Compare(ao, bo)
		}
		if stringValue(a["label"]) != stringValue(b["label"]) {
			return cmp.Compare(stringValue(a["label"]), stringValue(b["label"]))
		}
		return cmp.Compare(intValue(a["id"]), intValue(b["id"]))
	})
	return out
}

func buildDocumentsList(st *stores) []any {
	out := []any{}
	for _, docID := range st.documentIDs {
		doc := st.documentsByID[docID]
		updated := doc["updatedAtMs"]
		if updated == nil {
			updated = int64(0)
		}
		out = append(out, map[string]any{"id": doc["id"], "titleText": doc["titleText"], "updatedAtMs": updated})
	}
	slices.SortFunc(out, func(left, right any) int {
		a := left.(map[string]any)
		b := right.(map[string]any)
		au := int64Value(a["updatedAtMs"])
		bu := int64Value(b["updatedAtMs"])
		if au != bu {
			return cmp.Compare(bu, au)
		}
		return cmp.Compare(intValue(a["id"]), intValue(b["id"]))
	})
	return out
}

func buildContentCards(st *stores) []any {
	out := []any{}
	for _, contentID := range st.contentIDs {
		item := st.contentByID[contentID]
		tags := []map[string]any{}
		for _, tagID := range intSlice(item["tagIds"]) {
			if tag := st.tagsByID[tagID]; tag != nil {
				tags = append(tags, compactTag(tag))
			}
		}
		slices.SortFunc(tags, compareTags)
		out = append(out, map[string]any{"id": item["id"], "tags": eventsAny(tags), "title": item["title"]})
	}
	slices.SortFunc(out, func(left, right any) int {
		a := left.(map[string]any)
		b := right.(map[string]any)
		if stringValue(a["title"]) != stringValue(b["title"]) {
			return alphaCompare(stringValue(a["title"]), stringValue(b["title"]))
		}
		return cmp.Compare(intValue(a["id"]), intValue(b["id"]))
	})
	return out
}

func createSearchData(st *stores) []any {
	items := []any{}
	for _, personID := range st.peopleIDs {
		person := st.peopleByID[personID]
		text := person["name"]
		items = append(items, map[string]any{"id": person["id"], "norm": normalizeForSearch(text), "text": text, "type": "person"})
	}
	for _, contentID := range st.contentIDs {
		item := st.contentByID[contentID]
		text := item["title"]
		items = append(items, map[string]any{"id": item["id"], "norm": normalizeForSearch(text), "text": text, "type": "content"})
	}
	for _, orgID := range st.organizationIDs {
		org := st.organizationsByID[orgID]
		text := org["name"]
		items = append(items, map[string]any{"id": org["id"], "norm": normalizeForSearch(text), "text": text, "type": "organization"})
	}
	slices.SortStableFunc(items, func(left, right any) int {
		a := left.(map[string]any)
		b := right.(map[string]any)
		return alphaCompare(stringValue(a["text"]), stringValue(b["text"]))
	})
	return items
}

func buildTagIDsByLabel(tagTypes []hackertracker.TagType) map[string]any {
	byLabel := map[string]any{}
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
			existing, exists := byLabel[key]
			if !exists {
				byLabel[key] = id
				continue
			}
			existingID := intValue(existing)
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
	result := map[string]any{"byLabel": byLabel, "version": 1}
	if len(collisions) > 0 {
		collisionObj := map[string]any{}
		for key, set := range collisions {
			collisionObj[key] = slices.Sorted(maps.Keys(set))
		}
		result["collisions"] = collisionObj
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

func compactTag(tag map[string]any) map[string]any {
	return map[string]any{
		"colorBackground": tag["colorBackground"],
		"colorForeground": tag["colorForeground"],
		"id":              tag["id"],
		"label":           tag["label"],
	}
}

func stringValue(value any) string {
	return fmt.Sprint(value)
}

func intValue(value any) int {
	id, _ := normalizeID(value)
	return id
}

func int64Value(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case nil:
		return 0
	default:
		id, _ := normalizeID(v)
		return int64(id)
	}
}

func intSlice(value any) []int {
	switch v := value.(type) {
	case []int:
		return slices.Clone(v)
	case []any:
		out := []int{}
		for _, item := range v {
			if id, ok := normalizeID(item); ok {
				out = append(out, id)
			}
		}
		return out
	default:
		return nil
	}
}

func eventsAny[T any](items []T) []any {
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}

func nullableOrderValue(value any) *int {
	if value == nil {
		return nil
	}
	id, ok := normalizeID(value)
	if !ok {
		return nil
	}
	return &id
}

func nonEmptyString(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	text := fmt.Sprint(value)
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func joinComma(values []string) string {
	return strings.Join(values, ", ")
}
