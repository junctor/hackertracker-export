package export

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := StableJSON(value, sanitize)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

func WriteArtifacts(outDir string, artifacts Artifacts) ([]string, error) {
	dirs := []string{"entities", "indexes", "views", "details", "derived", "raw"}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		if err := os.RemoveAll(filepath.Join(outDir, dir)); err != nil {
			return nil, err
		}
	}
	for _, dir := range []string{"entities", "indexes", "views", "details", "derived"} {
		if err := os.MkdirAll(filepath.Join(outDir, dir), 0o755); err != nil {
			return nil, err
		}
	}

	written := []string{}
	write := func(rel string, value any) error {
		path := filepath.Join(outDir, rel)
		if err := WriteJSON(path, value, true); err != nil {
			return err
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
	if err := write(filepath.Join("derived", "tagIdsByLabel.json"), artifacts.Derived["tagIdsByLabel"]); err != nil {
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
		writeJSONString(buf, v)
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
		if float64(int64(v)) == v {
			buf.WriteString(strconv.FormatInt(int64(v), 10))
		} else {
			buf.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		}
	case json.Number:
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
		items := make([]any, len(v))
		for i := range v {
			items[i] = v[i]
		}
		return writeStable(buf, items)
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
			writeJSONString(buf, key)
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

func writeJSONString(buf *bytes.Buffer, value string) {
	var tmp bytes.Buffer
	enc := json.NewEncoder(&tmp)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		panic(err)
	}
	buf.Write(bytes.TrimRight(tmp.Bytes(), "\n"))
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
