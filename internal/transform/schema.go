package transform

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

func normalizeID(value any) (int, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return intFromInt64(v)
	case uint:
		return intFromUint64(uint64(v))
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return intFromUint64(uint64(v))
	case uint64:
		return intFromUint64(v)
	case float64:
		return intFromFloat64(v)
	case float32:
		return intFromFloat64(float64(v))
	case json.Number:
		return normalizeID(string(v))
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		if i, err := strconv.ParseInt(trimmed, 10, strconv.IntSize); err == nil {
			return int(i), true
		}
		if u, err := strconv.ParseUint(trimmed, 10, strconv.IntSize); err == nil {
			return intFromUint64(u)
		}
		f, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, false
		}
		return intFromFloat64(f)
	default:
		return normalizeID(fmt.Sprint(v))
	}
}

func normalizeOrder(value any) *int {
	id, ok := normalizeID(value)
	if !ok {
		return nil
	}
	return &id
}

func uniqueIDs[T any](ids []T, valid map[int]bool) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, raw := range ids {
		id, ok := normalizeID(raw)
		if !ok {
			continue
		}
		if valid != nil && !valid[id] {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func parseTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func isoTime(value string) string {
	t, ok := parseTime(value)
	if !ok {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

func timestampSeconds(value string) int64 {
	t, ok := parseTime(value)
	if !ok {
		return 0
	}
	return t.Unix()
}

func eventDay(value, timezone string) string {
	t, ok := parseTime(value)
	if !ok {
		return ""
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return ""
	}
	return t.In(loc).Format("2006-01-02")
}

func eventTimeTable(value string, showTZ bool, timezone string) string {
	t, ok := parseTime(value)
	if !ok {
		return ""
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return ""
	}
	if showTZ {
		return t.In(loc).Format("15:04 MST")
	}
	return t.In(loc).Format("15:04")
}

func resolveUpdatedAtMs(updatedAt any, updatedTSZ, updatedAtStr string) *int64 {
	if updatedAt != nil {
		switch v := updatedAt.(type) {
		case map[string]any:
			if seconds, ok := normalizeID(v["seconds"]); ok {
				ms := int64(seconds) * 1000
				return &ms
			}
		case string:
			if t, ok := parseTime(v); ok {
				ms := t.UnixMilli()
				return &ms
			}
		}
	}
	for _, value := range []string{updatedTSZ, updatedAtStr} {
		if t, ok := parseTime(value); ok {
			ms := t.UnixMilli()
			return &ms
		}
	}
	return nil
}

func sortEventIndex(index map[string][]int, eventStarts map[int]int64) {
	for key := range index {
		slices.SortStableFunc(index[key], func(a, b int) int {
			return cmp.Or(
				cmp.Compare(eventStarts[a], eventStarts[b]),
				cmp.Compare(a, b),
			)
		})
	}
}

func intFromInt64(value int64) (int, bool) {
	id := int(value)
	if int64(id) != value {
		return 0, false
	}
	return id, true
}

func intFromUint64(value uint64) (int, bool) {
	if value > uint64(math.MaxInt) {
		return 0, false
	}
	return int(value), true
}

func intFromFloat64(value float64) (int, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value {
		return 0, false
	}
	text := strconv.FormatFloat(value, 'f', 0, 64)
	id, err := strconv.ParseInt(text, 10, strconv.IntSize)
	if err != nil {
		return 0, false
	}
	return int(id), true
}
