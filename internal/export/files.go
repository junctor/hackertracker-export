package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var longSpacesRE = regexp.MustCompile(` {3,}`)

type Artifacts struct {
	Manifest any
	Entities map[string]any
	Indexes  map[string]any
	Views    map[string]any
	Derived  map[string]any
	Details  map[string]map[int]any
}

func StableJSON(value any, sanitize bool) ([]byte, error) {
	if sanitize {
		value = sanitizeStrings(value)
	}
	var buf bytes.Buffer
	if err := writeStable(&buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func WriteJSON(path string, value any, sanitize bool) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", dir, err)
	}
	data, err := StableJSON(value, sanitize)
	if err != nil {
		return fmt.Errorf("encode %q: %w", path, err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", path, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp file for %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file for %q: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("replace %q: %w", path, err)
	}
	return nil
}

func WriteArtifacts(outDir string, artifacts Artifacts) ([]string, error) {
	dirs := []string{"entities", "indexes", "views", "details", "derived", "raw"}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory %q: %w", outDir, err)
	}
	for _, dir := range dirs {
		path := filepath.Join(outDir, dir)
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("clear generated directory %q: %w", path, err)
		}
	}
	for _, dir := range []string{"entities", "indexes", "views", "details", "derived"} {
		path := filepath.Join(outDir, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, fmt.Errorf("create generated directory %q: %w", path, err)
		}
	}

	written := []string{}
	write := func(rel string, value any) error {
		path := filepath.Join(outDir, rel)
		if err := WriteJSON(path, value, true); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		written = append(written, path)
		return nil
	}

	if err := write("manifest.json", artifacts.Manifest); err != nil {
		return nil, err
	}
	for _, name := range requiredEntityNames() {
		value, ok := artifacts.Entities[name]
		if !ok {
			return nil, fmt.Errorf("missing generated artifact: %s", name)
		}
		if err := write(filepath.Join("entities", name+".json"), value); err != nil {
			return nil, err
		}
	}
	for _, name := range []string{"eventsByDay", "eventsByTag"} {
		value, ok := artifacts.Indexes[name]
		if !ok {
			return nil, fmt.Errorf("missing generated artifact: %s", name)
		}
		if err := write(filepath.Join("indexes", name+".json"), value); err != nil {
			return nil, err
		}
	}
	for _, name := range requiredViewNames() {
		value, ok := artifacts.Views[name]
		if !ok {
			return nil, fmt.Errorf("missing generated artifact: %s", name)
		}
		if err := write(filepath.Join("views", name+".json"), value); err != nil {
			return nil, err
		}
	}
	tagIDsByLabel, ok := artifacts.Derived["tagIdsByLabel"]
	if !ok {
		return nil, fmt.Errorf("missing generated artifact: tagIdsByLabel")
	}
	if err := write(filepath.Join("derived", "tagIdsByLabel.json"), tagIDsByLabel); err != nil {
		return nil, err
	}

	groups := make([]string, 0, len(artifacts.Details))
	for group := range artifacts.Details {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	for _, group := range groups {
		ids := make([]int, 0, len(artifacts.Details[group]))
		for id := range artifacts.Details[group] {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			if err := write(filepath.Join("details", group, strconv.Itoa(id)+".json"), artifacts.Details[group][id]); err != nil {
				return nil, err
			}
		}
	}
	sort.Strings(written)
	return written, nil
}

func requiredEntityNames() []string {
	return []string{"articles", "content", "documents", "events", "locations", "organizations", "people", "tags"}
}

func requiredViewNames() []string {
	return []string{
		"announcementsList",
		"bookmarkEventsById",
		"contentCards",
		"documentsList",
		"locationCards",
		"organizationsCards",
		"peopleCards",
		"searchData",
		"scheduleDays",
		"tagTypesBrowse",
	}
}

func sanitizeStrings(value any) any {
	switch v := value.(type) {
	case string:
		return sanitizeString(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = sanitizeStrings(item)
		}
		return out
	case []int:
		return v
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = sanitizeStrings(item)
		}
		return out
	case map[int]any:
		out := make(map[int]any, len(v))
		for key, item := range v {
			out[key] = sanitizeStrings(item)
		}
		return out
	default:
		return value
	}
}

func sanitizeString(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\u2028", "\n")
	value = strings.ReplaceAll(value, "\u2029", "\n")
	value = strings.ReplaceAll(value, "\t", " ")
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		line = longSpacesRE.ReplaceAllString(line, " ")
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

func writeStable(buf *bytes.Buffer, value any) error {
	switch v := value.(type) {
	case nil:
		buf.WriteString("null")
	case string:
		return writeJSONString(buf, v)
	case bool:
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case int:
		buf.WriteString(strconv.Itoa(v))
	case int64:
		buf.WriteString(strconv.FormatInt(v, 10))
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("unsupported floating-point value %v", v)
		}
		buf.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
	case json.Number:
		if !json.Valid([]byte(v.String())) {
			return fmt.Errorf("unsupported JSON number %q", v.String())
		}
		buf.WriteString(v.String())
	case []int:
		buf.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(strconv.Itoa(item))
		}
		buf.WriteByte(']')
	case []string:
		buf.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeJSONString(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case []any:
		buf.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeStable(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[int]any:
		m := make(map[string]any, len(v))
		for key, item := range v {
			m[strconv.Itoa(key)] = item
		}
		return writeStable(buf, m)
	case map[string]int:
		m := make(map[string]any, len(v))
		for key, item := range v {
			m[key] = item
		}
		return writeStable(buf, m)
	case map[string][]int:
		m := make(map[string]any, len(v))
		for key, item := range v {
			m[key] = item
		}
		return writeStable(buf, m)
	case map[string]any:
		keys := objectKeys(v)
		buf.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeJSONString(buf, key); err != nil {
				return err
			}
			buf.WriteByte(':')
			if err := writeStable(buf, v[key]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		var decoded any
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		if err := dec.Decode(&decoded); err != nil {
			return err
		}
		return writeStable(buf, decoded)
	}
	return nil
}

func writeJSONString(buf *bytes.Buffer, value string) error {
	var tmp bytes.Buffer
	enc := json.NewEncoder(&tmp)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return err
	}
	buf.Write(bytes.TrimRight(tmp.Bytes(), "\n"))
	return nil
}

func objectKeys(m map[string]any) []string {
	intKeys := []int{}
	intKeyByText := map[int]string{}
	stringKeys := []string{}
	for key := range m {
		if n, ok := arrayIndexKey(key); ok {
			intKeys = append(intKeys, n)
			intKeyByText[n] = key
			continue
		}
		stringKeys = append(stringKeys, key)
	}
	sort.Ints(intKeys)
	sort.Strings(stringKeys)
	out := make([]string, 0, len(m))
	for _, key := range intKeys {
		out = append(out, intKeyByText[key])
	}
	out = append(out, stringKeys...)
	return out
}

func arrayIndexKey(key string) (int, bool) {
	if key == "" {
		return 0, false
	}
	if len(key) > 1 && key[0] == '0' {
		return 0, false
	}
	n, err := strconv.Atoi(key)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}
