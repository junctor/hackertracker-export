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
	Manifest     any
	Conference   any
	Schedule     ScheduleExport
	Entities     map[string]any
	Indexes      map[string]any
	Views        map[string]any
	Details      map[string]map[int]any
	DetailShards map[string]DetailShardSpec
}

type DetailShardSpec struct {
	Count  int
	Digits int
}

var generatedDirs = [...]string{"views", "details", "exports"}

var prunedDirs = [...]string{"raw", "entities", "indexes", "derived"}

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
	for _, dir := range prunedDirs {
		path := filepath.Join(outDir, dir)
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("clear unpublished directory %q: %w", path, err)
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

	if err := write("conference.json", artifacts.Conference); err != nil {
		return nil, err
	}
	if err := write(filepath.Join("exports", "schedule.json"), artifacts.Schedule); err != nil {
		return nil, err
	}
	scheduleCSVPath := filepath.Join(outDir, "exports", "schedule.csv")
	if err := writeScheduleCSV(scheduleCSVPath, artifacts.Schedule); err != nil {
		return nil, fmt.Errorf("write exports/schedule.csv: %w", err)
	}
	written = append(written, scheduleCSVPath)
	for _, name := range []string{
		"announcementsList",
		"contentCards",
		"contentFilterIndex",
		"documentsList",
		"locationCards",
		"organizationsBrowse",
		"peopleCards",
		"searchData",
		"scheduleBrowse",
		"scheduleFilterIndex",
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
	groups := slices.Sorted(maps.Keys(artifacts.DetailShards))
	for _, group := range groups {
		spec := artifacts.DetailShards[group]
		if spec.Count <= 0 || spec.Digits <= 0 {
			return nil, fmt.Errorf("invalid detail shard spec for %s", group)
		}
		values, ok := artifacts.Details[group]
		if !ok {
			return nil, fmt.Errorf("missing detail values for %s", group)
		}
		shards := make([]map[string]any, spec.Count)
		for index := range shards {
			shards[index] = map[string]any{}
		}
		for _, id := range slices.Sorted(maps.Keys(values)) {
			index := ((id % spec.Count) + spec.Count) % spec.Count
			shards[index][strconv.Itoa(id)] = values[id]
		}
		for index, shard := range shards {
			name := fmt.Sprintf("%0*d.json", spec.Digits, index)
			if err := write(filepath.Join("details", group, name), shard); err != nil {
				return nil, err
			}
		}
	}
	// Write the cache-invalidation marker only after every referenced artifact
	// has been generated successfully.
	if err := write("manifest.json", artifacts.Manifest); err != nil {
		return nil, err
	}
	slices.Sort(written)
	return written, nil
}
