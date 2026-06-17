package hackertracker

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
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
	if err := decodeCollection(raw["events"], &data.Events); err != nil {
		return data, fmt.Errorf("events: %w", err)
	}
	if err := decodeCollection(raw["locations"], &data.Locations); err != nil {
		return data, fmt.Errorf("locations: %w", err)
	}
	data.Menus = raw["menus"]
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
	normalized := normalizeFirestoreValue(items)
	b, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

func normalizeFirestoreValue(value any) any {
	switch v := value.(type) {
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano)
	case []map[string]any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = normalizeFirestoreValue(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = normalizeFirestoreValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = normalizeFirestoreValue(item)
		}
		return out
	default:
		if isScalar(value) {
			return value
		}
		return fmt.Sprint(value)
	}
}

func isScalar(value any) bool {
	switch value.(type) {
	case nil, bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return true
	}
	switch rv.Kind() {
	case reflect.Pointer, reflect.Struct, reflect.Interface:
		return false
	default:
		return true
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
		key, err := collectionSortKey(item)
		if err != nil {
			return fmt.Errorf("document %d: %w", i, err)
		}
		sortable[i].key = key
	}

	sort.SliceStable(sortable, func(i, j int) bool {
		if sortable[i].hasID && sortable[j].hasID {
			return sortable[i].id < sortable[j].id
		}
		if sortable[i].hasID {
			return true
		}
		if sortable[j].hasID {
			return false
		}
		return sortable[i].key < sortable[j].key
	})
	for i := range sortable {
		items[i] = sortable[i].item
	}
	return nil
}

func collectionSortKey(item map[string]any) (string, error) {
	data, err := json.Marshal(item)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func idString(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "", false
	case string:
		if v == "" {
			return "", false
		}
		return v, true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	default:
		return fmt.Sprint(v), true
	}
}
