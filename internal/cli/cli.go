package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/junctor/hackertracker-export/internal/export"
	"github.com/junctor/hackertracker-export/internal/transform"
	"github.com/junctor/hackertracker-export/pkg/hackertracker"
)

func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return nil
	}
	switch args[0] {
	case "conferences":
		return runConferences(stdout)
	case "fetch":
		return runFetch(args[1:], stdout, stderr)
	case "info-export":
		return runInfoExport(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runConferences(stdout io.Writer) error {
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
	return encodeJSON(stdout, confs)
}

func runFetch(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	conference := fs.String("conference", "", "conference code")
	outDir := fs.String("out", "", "output directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if *conference == "" {
		return fmt.Errorf("missing --conference")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client, err := hackertracker.NewClient(ctx)
	if err != nil {
		return err
	}
	conf, data, err := fetchRaw(ctx, client, *conference)
	if err != nil {
		return err
	}
	if *outDir == "" {
		return encodeJSON(stdout, map[string]any{"conference": conf, "collections": data})
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", *outDir, err)
	}
	if err := export.WriteJSON(filepath.Join(*outDir, "conference.json"), conf); err != nil {
		return fmt.Errorf("write conference metadata: %w", err)
	}
	count := 1
	for _, name := range hackertracker.CollectionNames() {
		if err := export.WriteJSON(filepath.Join(*outDir, name+".json"), data[name]); err != nil {
			return fmt.Errorf("write raw collection %q: %w", name, err)
		}
		count++
	}
	fmt.Fprintf(stdout, "Wrote %d raw files to %s\n", count, *outDir)
	return nil
}

func runInfoExport(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("info-export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	conference := fs.String("conference", "", "conference code")
	conferenceShort := fs.String("c", "", "conference code")
	outDir := fs.String("out", "", "output directory")
	outShort := fs.String("o", "", "output directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	confCode := firstNonEmpty(*conference, *conferenceShort)
	if confCode == "" && fs.NArg() > 0 {
		confCode = fs.Arg(0)
	}
	if confCode == "" {
		printInfoExportHelp(stdout)
		return fmt.Errorf("please provide a conference code")
	}
	out := firstNonEmpty(*outDir, *outShort)
	if out == "" {
		out = filepath.Join(".", "out", "ht", strings.ToLower(confCode))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client, err := hackertracker.NewClient(ctx)
	if err != nil {
		return err
	}
	conf, data, _, err := client.SourceData(ctx, confCode)
	if err != nil {
		return fmt.Errorf("load source data for %q: %w", confCode, err)
	}
	artifacts, err := transform.Build(conf, data, transform.BuildOptions{
		SchemaVersion:  2,
		BuildTimestamp: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("build export artifacts for %q: %w", conf.Code, err)
	}
	written, err := export.WriteArtifacts(out, artifacts)
	if err != nil {
		return fmt.Errorf("write export artifacts to %q: %w", out, err)
	}
	fmt.Fprintf(stdout, "Exported %s -> %s\n", conf.Code, out)
	fmt.Fprintf(stdout, "Wrote %d files\n", len(written))
	return nil
}

func fetchRaw(ctx context.Context, client *hackertracker.Client, conference string) (hackertracker.Conference, map[string][]map[string]any, error) {
	conf, err := client.Conference(ctx, conference)
	if err != nil {
		return hackertracker.Conference{}, nil, err
	}
	raw := map[string][]map[string]any{}
	fetchCode := conf.Code
	if fetchCode == "" {
		fetchCode = conference
	}
	for _, name := range hackertracker.CollectionNames() {
		items, err := client.Collection(ctx, fetchCode, name)
		if err != nil {
			return hackertracker.Conference{}, nil, fmt.Errorf("fetch %s for %q: %w", name, fetchCode, err)
		}
		raw[name] = items
	}
	return conf, raw, nil
}

func encodeJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	return enc.Encode(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  hackertracker conferences
  hackertracker fetch --conference <code> [--out <dir>]
  hackertracker info-export --conference <code> --out <dir>

Examples:
  hackertracker conferences
  hackertracker fetch --conference defcon34 --out ./raw
  hackertracker info-export --conference defcon34 --out ./public/defcon34/data`)
}

func printInfoExportHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  hackertracker info-export --conference <code> --out <dir>

Options:
  --conference, -c <code>  Conference code
  --out, -o <dir>          Output directory`)
}
