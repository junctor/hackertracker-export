package export

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
)

type Artifacts struct {
	Manifest any
	Entities map[string]any
	Indexes  map[string]any
	Views    map[string]any
	Derived  map[string]any
	Details  map[string]map[int]any
}

var generatedDirs = [...]string{"entities", "indexes", "views", "details", "derived"}

func writeJSON(path string, value any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", dir, err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode %q: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

func WriteArtifacts(outDir string, artifacts Artifacts) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory %q: %w", outDir, err)
	}
	for _, dir := range generatedDirs {
		path := filepath.Join(outDir, dir)
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("clear generated directory %q: %w", path, err)
		}
	}
	for _, dir := range generatedDirs {
		path := filepath.Join(outDir, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, fmt.Errorf("create generated directory %q: %w", path, err)
		}
	}

	written := []string{}
	write := func(rel string, value any) error {
		path := filepath.Join(outDir, rel)
		if err := writeJSON(path, value); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		written = append(written, path)
		return nil
	}

	if err := write("manifest.json", artifacts.Manifest); err != nil {
		return nil, err
	}
	for _, name := range []string{"articles", "content", "documents", "locations", "organizations", "people", "sessions", "tags", "tagTypes"} {
		value, ok := artifacts.Entities[name]
		if !ok {
			return nil, fmt.Errorf("missing generated artifact: %s", name)
		}
		if err := write(filepath.Join("entities", name+".json"), value); err != nil {
			return nil, err
		}
	}
	for _, name := range []string{"sessionsByDay", "sessionsByTag"} {
		value, ok := artifacts.Indexes[name]
		if !ok {
			return nil, fmt.Errorf("missing generated artifact: %s", name)
		}
		if err := write(filepath.Join("indexes", name+".json"), value); err != nil {
			return nil, err
		}
	}
	for _, name := range []string{
		"announcementsList",
		"bookmarkSessionsById",
		"contentCards",
		"documentsList",
		"locationCards",
		"organizationsCards",
		"peopleCards",
		"searchData",
		"scheduleDays",
		"tagTypesBrowse",
	} {
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

	groups := slices.Sorted(maps.Keys(artifacts.Details))
	for _, group := range groups {
		ids := slices.Sorted(maps.Keys(artifacts.Details[group]))
		for _, id := range ids {
			if err := write(filepath.Join("details", group, strconv.Itoa(id)+".json"), artifacts.Details[group][id]); err != nil {
				return nil, err
			}
		}
	}
	slices.Sort(written)
	return written, nil
}
