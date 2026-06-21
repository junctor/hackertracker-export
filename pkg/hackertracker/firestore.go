package hackertracker

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

func DecodeSourceData(raw map[string][]map[string]any) (SourceData, error) {
	var data SourceData
	if err := decodeCollection(raw["articles"], &data.Articles); err != nil {
		return data, fmt.Errorf("articles: %w", err)
	}
	if err := decodeCollection(raw["content"], &data.Content); err != nil {
		return data, fmt.Errorf("content: %w", err)
	}
	if err := decodeCollection(raw["documents"], &data.Documents); err != nil {
		return data, fmt.Errorf("documents: %w", err)
	}
	if err := decodeCollection(raw["locations"], &data.Locations); err != nil {
		return data, fmt.Errorf("locations: %w", err)
	}
	if err := decodeCollection(raw["menus"], &data.Menus); err != nil {
		return data, fmt.Errorf("menus: %w", err)
	}
	if err := decodeCollection(raw["organizations"], &data.Organizations); err != nil {
		return data, fmt.Errorf("organizations: %w", err)
	}
	if err := decodeCollection(raw["speakers"], &data.Speakers); err != nil {
		return data, fmt.Errorf("speakers: %w", err)
	}
	if err := decodeCollection(raw["tagtypes"], &data.TagTypes); err != nil {
		return data, fmt.Errorf("tagtypes: %w", err)
	}
	return data, nil
}

func decodeCollection[T any](items []map[string]any, dest *[]T) error {
	b, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

func rawFirestoreValue(value any) any {
	switch v := value.(type) {
	case time.Time:
		return map[string]any{
			"nanoseconds": v.Nanosecond(),
			"seconds":     v.Unix(),
			"type":        "firestore/timestamp/1.0",
		}
	case []map[string]any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = rawFirestoreValue(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = rawFirestoreValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = rawFirestoreValue(item)
		}
		return out
	default:
		return value
	}
}

func sortCollection(items []map[string]any) error {
	type sortableItem struct {
		item  map[string]any
		id    string
		hasID bool
		key   string
	}

	sortable := make([]sortableItem, len(items))
	for i, item := range items {
		id, ok := idString(item["id"])
		sortable[i] = sortableItem{item: item, id: id, hasID: ok}
		if ok {
			continue
		}
		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("document %d: %w", i, err)
		}
		sortable[i].key = string(data)
	}

	slices.SortStableFunc(sortable, func(a, b sortableItem) int {
		if a.hasID != b.hasID {
			if a.hasID {
				return -1
			}
			return 1
		}
		return cmp.Or(
			cmp.Compare(a.id, b.id),
			cmp.Compare(a.key, b.key),
		)
	})
	for i := range sortable {
		items[i] = sortable[i].item
	}
	return nil
}

func idString(value any) (string, bool) {
	if value == nil {
		return "", false
	}
	text := fmt.Sprint(value)
	return text, text != "" && text != "<nil>"
}
