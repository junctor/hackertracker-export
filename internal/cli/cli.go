package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/junctor/hackertracker-export/internal/export"
	"github.com/junctor/hackertracker-export/internal/transform"
	"github.com/junctor/hackertracker-export/pkg/hackertracker"
)

func Run(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printUsage()
		return nil
	}
	switch args[0] {
	case "conferences":
		return runConferences()
	case "fetch":
		return runFetch(args[1:])
	case "info":
		return runInfoExport(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runConferences() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client, err := hackertracker.NewClient(ctx)
	if err != nil {
		return err
	}
	confs, err := client.Conferences(ctx)
	if err != nil {
		return fmt.Errorf("fetch conferences: %w", err)
	}
	return json.NewEncoder(os.Stdout).Encode(confs)
}

func runFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	conference := fs.String("conference", "", "conference code")
	outDir := fs.String("out", "", "output directory")
	document := fs.String("document", "", "raw document name")
	collection := fs.String("collection", "", "raw collection name")
	stdout := fs.Bool("stdout", false, "print JSON to stdout")
	all := fs.Bool("all", false, "fetch every raw document and collection")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	*conference = strings.TrimSpace(*conference)
	if *conference == "" {
		fs.Usage()
		return fmt.Errorf("missing --conference")
	}

	targetKind, targetName := "", ""
	targets := 0
	setTarget := func(kind, name string) {
		targetKind = kind
		targetName = strings.TrimSpace(name)
		targets++
	}
	if *all {
		setTarget("all", "all")
	}
	if strings.TrimSpace(*document) != "" {
		setTarget("document", *document)
	}
	if strings.TrimSpace(*collection) != "" {
		setTarget("collection", *collection)
	}
	for _, arg := range fs.Args() {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		switch {
		case strings.EqualFold(arg, "all"):
			setTarget("all", "all")
		case strings.EqualFold(arg, "conference"):
			setTarget("document", "conference")
		default:
			setTarget("collection", arg)
		}
	}
	if targets > 1 {
		return fmt.Errorf("choose only one of --document, --collection, --all, or a positional target")
	}
	if targetName == "" {
		var err error
		targetKind, targetName, err = chooseRawFetchTarget()
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client, err := hackertracker.NewClient(ctx)
	if err != nil {
		return err
	}
	conf, err := client.RawConference(ctx, *conference)
	if err != nil {
		return err
	}
	fetchCode, _ := conf["code"].(string)
	if fetchCode == "" {
		fetchCode = *conference
	}
	rawOutDir := strings.TrimSpace(*outDir)
	if rawOutDir == "" {
		rawOutDir = filepath.Join(".", "out", "ht", strings.ToLower(*conference), "raw")
	}

	if targetKind == "all" {
		data := map[string][]map[string]any{}
		for _, name := range hackertracker.CollectionNames() {
			items, err := client.RawCollection(ctx, fetchCode, name)
			if err != nil {
				return fmt.Errorf("fetch %s for %q: %w", name, fetchCode, err)
			}
			data[name] = items
		}
		if *stdout {
			return json.NewEncoder(os.Stdout).Encode(map[string]any{"conference": conf, "collections": data})
		}
		if err := writeRawJSON(filepath.Join(rawOutDir, "conference.json"), conf); err != nil {
			return fmt.Errorf("write conference metadata: %w", err)
		}
		count := 1
		for _, name := range hackertracker.CollectionNames() {
			if err := writeRawJSON(filepath.Join(rawOutDir, name+".json"), data[name]); err != nil {
				return fmt.Errorf("write raw collection %q: %w", name, err)
			}
			count++
		}
		fmt.Printf("Wrote %d raw files to %s\n", count, rawOutDir)
		return nil
	}

	var value any
	if targetKind == "document" {
		if targetName != "conference" {
			return fmt.Errorf("unknown raw document %q", targetName)
		}
		value = conf
	} else {
		value, err = client.RawCollection(ctx, fetchCode, targetName)
		if err != nil {
			return fmt.Errorf("fetch %s for %q: %w", targetName, fetchCode, err)
		}
	}

	if *stdout {
		return json.NewEncoder(os.Stdout).Encode(value)
	}
	path := filepath.Join(rawOutDir, targetName+".json")
	if err := writeRawJSON(path, value); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n", path)
	return nil
}

func runInfoExport(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	var conferences []string
	fs.Func("conference", "conference code, repeatable", func(value string) error {
		conferences = append(conferences, value)
		return nil
	})
	outDir := fs.String("out", "", "output directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	conferences = append(conferences, fs.Args()...)
	cleaned := conferences[:0]
	for _, conference := range conferences {
		if conference = strings.TrimSpace(conference); conference != "" {
			cleaned = append(cleaned, conference)
		}
	}
	conferences = cleaned
	if len(conferences) == 0 {
		fs.Usage()
		return fmt.Errorf("missing --conference")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client, err := hackertracker.NewClient(ctx)
	if err != nil {
		return err
	}
	outRoot := strings.TrimSpace(*outDir)
	multiple := len(conferences) > 1
	for _, confCode := range conferences {
		out := outRoot
		if out == "" {
			out = filepath.Join(".", "out", "ht", strings.ToLower(confCode))
		} else if multiple {
			out = filepath.Join(out, strings.ToLower(confCode))
		}
		conf, data, err := client.SourceData(ctx, confCode)
		if err != nil {
			return fmt.Errorf("load source data for %q: %w", confCode, err)
		}
		artifacts, err := transform.Build(conf, data)
		if err != nil {
			return fmt.Errorf("build export artifacts for %q: %w", conf.Code, err)
		}
		written, err := export.WriteArtifacts(out, artifacts)
		if err != nil {
			return fmt.Errorf("write export artifacts to %q: %w", out, err)
		}
		fmt.Printf("Exported %s -> %s\n", conf.Code, out)
		fmt.Printf("Wrote %d files\n", len(written))
	}
	return nil
}

func chooseRawFetchTarget() (string, string, error) {
	type target struct {
		kind string
		name string
	}

	targets := []target{{kind: "document", name: "conference"}}
	for _, name := range hackertracker.CollectionNames() {
		targets = append(targets, target{kind: "collection", name: name})
	}
	targets = append(targets, target{kind: "all", name: "all"})

	fmt.Println("Available raw Firebase documents and collections:")
	for i, target := range targets {
		fmt.Printf("  %d) %s (%s)\n", i+1, target.name, target.kind)
	}
	fmt.Print("Choose one: ")

	var input string
	if _, err := fmt.Fscan(bufio.NewReader(os.Stdin), &input); err != nil {
		return "", "", fmt.Errorf("read selection: %w", err)
	}
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= len(targets) {
		target := targets[choice-1]
		return target.kind, target.name, nil
	}
	for _, target := range targets {
		if strings.EqualFold(input, target.name) {
			return target.kind, target.name, nil
		}
	}
	return "", "", fmt.Errorf("unknown raw Firebase selection %q", input)
}

func writeRawJSON(path string, value any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", dir, err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %q: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

func printUsage() {
	fmt.Println(`Usage:
  hackertracker conferences
  hackertracker fetch --conference <code> [--document conference|--collection <name>|--all] [--stdout] [--out <dir>]
  hackertracker info [--out <dir>] --conference <code> [<code>...]

Examples:
  hackertracker conferences
  hackertracker fetch --conference DEFCON34
  hackertracker fetch --conference DEFCON34 --collection events
  hackertracker fetch --conference DEFCON34 --document conference --stdout
  hackertracker fetch --conference DEFCON34 --all
  hackertracker info --conference defcon34 --out ./public/defcon34/data
  hackertracker info --out ./public --conference DCSG2026 DEFCON34 DEFCON33`)
}
