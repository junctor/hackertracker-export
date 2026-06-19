package export

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func NormalizeID(value any) (int, bool) {
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
		return NormalizeID(string(v))
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
		return NormalizeID(fmt.Sprint(v))
	}
}

func NormalizeOrder(value any) *int {
	id, ok := NormalizeID(value)
	if !ok {
		return nil
	}
	return &id
}

func UniqueIDs[T any](ids []T, valid map[int]bool) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, raw := range ids {
		id, ok := NormalizeID(raw)
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

func ParseTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z07:00"}
	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func ISOTime(value string) string {
	t, ok := ParseTime(value)
	if !ok {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

func TimestampSeconds(value string) int64 {
	t, ok := ParseTime(value)
	if !ok {
		return 0
	}
	return t.Unix()
}

func EventDay(value, timezone string) string {
	t, ok := ParseTime(value)
	if !ok {
		return ""
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return ""
	}
	return t.In(loc).Format("2006-01-02")
}

func EventTimeTable(value string, showTZ bool, timezone string) string {
	t, ok := ParseTime(value)
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

func ResolveUpdatedAtMs(updatedAt any, updatedTSZ, updatedAtStr string) *int64 {
	if updatedAt != nil {
		switch v := updatedAt.(type) {
		case map[string]any:
			if seconds, ok := NormalizeID(v["seconds"]); ok {
				ms := int64(seconds) * 1000
				return &ms
			}
		case string:
			if t, ok := ParseTime(v); ok {
				ms := t.UnixMilli()
				return &ms
			}
		}
	}
	for _, value := range []string{updatedTSZ, updatedAtStr} {
		if t, ok := ParseTime(value); ok {
			ms := t.UnixMilli()
			return &ms
		}
	}
	return nil
}

func intFromInt64(value int64) (int, bool) {
	id := int(value)
	if int64(id) != value {
		return 0, false
	}
	return id, true
}

func intFromUint64(value uint64) (int, bool) {
	maxInt := uint64(^uint(0) >> 1)
	if value > maxInt {
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
