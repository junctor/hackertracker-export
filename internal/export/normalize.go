package export

import (
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
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		if v > math.MaxInt {
			return 0, false
		}
		return int(v), true
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return int(v), true
	case float32:
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false
		}
		return int(f), true
	case jsonNumber:
		return NormalizeID(string(v))
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(trimmed, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false
		}
		return int(f), true
	default:
		return NormalizeID(fmt.Sprint(v))
	}
}

type jsonNumber string

func NormalizeOrder(value any) *int {
	id, ok := NormalizeID(value)
	if !ok {
		return nil
	}
	return &id
}

func UniqueIDs(ids []any, valid map[int]bool) []int {
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
