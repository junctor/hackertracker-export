package cli

import (
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

var rawCollections = []string{
	"articles",
	"content",
	"documents",
	"locations",
	"organizations",
	"people",
	"sessions",
	"tags",
	"tagTypes",
}

type fetchOptions struct {
	conference string
	outDir     string
	stdout     bool
}

func Run(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printUsage()
		return nil
	}

	switch args[0] {
	case "conferences":
		return runConferences(args[1:])
	case "fetch":
		return runFetch(args[1:])
	case "info":
		return runInfoExport(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runConferences(args []string) error {
	fs := flag.NewFlagSet("conferences", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage:
  hackertracker conferences

Print available HackerTracker conferences as JSON.
`)
	}

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

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
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printFetchUsage()
		return nil
	}

	switch args[0] {
	case "conference":
		return runFetchConference(args[1:])
	case "articles":
		return runFetchCollection("articles", args[1:])
	case "content":
		return runFetchCollection("content", args[1:])
	case "documents":
		return runFetchCollection("documents", args[1:])
	case "locations":
		return runFetchCollection("locations", args[1:])
	case "organizations":
		return runFetchCollection("organizations", args[1:])
	case "people":
		return runFetchCollection("people", args[1:])
	case "sessions":
		return runFetchCollection("sessions", args[1:])
	case "tags":
		return runFetchCollection("tags", args[1:])
	case "tagTypes":
		return runFetchCollection("tagTypes", args[1:])
	case "all":
		return runFetchAll(args[1:])
	default:
		printFetchUsage()
		return fmt.Errorf("unknown fetch target %q", args[0])
	}
}

func runFetchConference(args []string) error {
	opts, err := parseFetchOptions("fetch conference", args)
	if err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := hackertracker.NewClient(ctx)
	if err != nil {
		return err
	}

	conf, err := client.RawConference(ctx, opts.conference)
	if err != nil {
		return err
	}

	return writeOrPrintRaw("conference", conf, rawOutputDir(opts), opts.stdout)
}

func runFetchCollection(name string, args []string) error {
	opts, err := parseFetchOptions("fetch "+name, args)
	if err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := hackertracker.NewClient(ctx)
	if err != nil {
		return err
	}

	conf, err := client.RawConference(ctx, opts.conference)
	if err != nil {
		return err
	}

	fetchCode := conferenceFetchCode(conf, opts.conference)

	value, err := client.RawCollection(ctx, fetchCode, name)
	if err != nil {
		return fmt.Errorf("fetch %s for %q: %w", name, fetchCode, err)
	}

	return writeOrPrintRaw(name, value, rawOutputDir(opts), opts.stdout)
}

func runFetchAll(args []string) error {
	opts, err := parseFetchOptions("fetch all", args)
	if err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := hackertracker.NewClient(ctx)
	if err != nil {
		return err
	}

	conf, err := client.RawConference(ctx, opts.conference)
	if err != nil {
		return err
	}

	fetchCode := conferenceFetchCode(conf, opts.conference)
	rawOutDir := rawOutputDir(opts)

	collections := map[string][]map[string]any{}

	for _, name := range rawCollections {
		items, err := client.RawCollection(ctx, fetchCode, name)
		if err != nil {
			return fmt.Errorf("fetch %s for %q: %w", name, fetchCode, err)
		}
		collections[name] = items
	}

	if opts.stdout {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"conference":  conf,
			"collections": collections,
		})
	}

	if err := writeRawJSON(filepath.Join(rawOutDir, "conference.json"), conf); err != nil {
		return fmt.Errorf("write conference metadata: %w", err)
	}

	count := 1
	for _, name := range rawCollections {
		if err := writeRawJSON(filepath.Join(rawOutDir, name+".json"), collections[name]); err != nil {
			return fmt.Errorf("write raw collection %q: %w", name, err)
		}
		count++
	}

	fmt.Printf("Wrote %d raw files to %s\n", count, rawOutDir)
	return nil
}

func parseFetchOptions(command string, args []string) (fetchOptions, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)

	var opts fetchOptions
	fs.StringVar(&opts.conference, "conference", "", "conference code")
	fs.StringVar(&opts.outDir, "out", "", "output directory")
	fs.BoolVar(&opts.stdout, "stdout", false, "print JSON to stdout")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage:
  hackertracker %s --conference <code> [--stdout] [--out <dir>]

Options:
`, command)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	if fs.NArg() > 0 {
		return opts, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	opts.conference = strings.TrimSpace(opts.conference)
	opts.outDir = strings.TrimSpace(opts.outDir)

	if opts.conference == "" {
		fs.Usage()
		return opts, fmt.Errorf("missing --conference")
	}

	return opts, nil
}

func runInfoExport(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)

	var conferences []string
	fs.Func("conference", "conference code, repeatable", func(value string) error {
		conferences = append(conferences, value)
		return nil
	})

	outDir := fs.String("out", "", "output directory")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage:
  hackertracker info [--out <dir>] --conference <code> [--conference <code>]
  hackertracker info [--out <dir>] --conference <code> [<code>...]

Options:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	conferences = append(conferences, fs.Args()...)
	conferences = cleanStrings(conferences)

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

func cleanStrings(values []string) []string {
	cleaned := values[:0]

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}

	return cleaned
}

func conferenceFetchCode(conf map[string]any, fallback string) string {
	code, _ := conf["code"].(string)
	code = strings.TrimSpace(code)

	if code == "" {
		return fallback
	}

	return code
}

func rawOutputDir(opts fetchOptions) string {
	if opts.outDir != "" {
		return opts.outDir
	}

	return filepath.Join(".", "out", "ht", strings.ToLower(opts.conference), "raw")
}

func writeOrPrintRaw(name string, value any, rawOutDir string, stdout bool) error {
	if stdout {
		return json.NewEncoder(os.Stdout).Encode(value)
	}

	path := filepath.Join(rawOutDir, name+".json")
	if err := writeRawJSON(path, value); err != nil {
		return err
	}

	fmt.Printf("Wrote %s\n", path)
	return nil
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
  hackertracker fetch <target> --conference <code> [--stdout] [--out <dir>]
  hackertracker info [--out <dir>] --conference <code> [--conference <code>]

Fetch targets:
  conference
  articles
  content
  documents
  locations
  organizations
  people
  sessions
  tags
  tagTypes
  all

Examples:
  hackertracker conferences
  hackertracker fetch conference --conference DEFCON34 --stdout
  hackertracker fetch content --conference DEFCON34
  hackertracker fetch sessions --conference DEFCON34
  hackertracker fetch tagTypes --conference DEFCON34
  hackertracker fetch all --conference DEFCON34
  hackertracker info --conference DEFCON34 --out ./out/ht/defcon34
  hackertracker info --out ./out/ht --conference DCSG2026 --conference DEFCON34`)
}

func printFetchUsage() {
	fmt.Println(`Usage:
  hackertracker fetch <target> --conference <code> [--stdout] [--out <dir>]

Targets:
  conference
  articles
  content
  documents
  locations
  organizations
  people
  sessions
  tags
  tagTypes
  all

Examples:
  hackertracker fetch conference --conference DEFCON34 --stdout
  hackertracker fetch content --conference DEFCON34
  hackertracker fetch sessions --conference DEFCON34
  hackertracker fetch all --conference DEFCON34

Target help:
  hackertracker fetch content -h
  hackertracker fetch all -h`)
}
