package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/junctor/hackertracker-export/internal/export"
	"github.com/junctor/hackertracker-export/pkg/hackertracker"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return nil
	}
	switch args[0] {
	case "conferences":
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
		data, err := json.Marshal(confs)
		if err != nil {
			return fmt.Errorf("encode conferences: %w", err)
		}
		fmt.Println(string(data))
		return nil
	case "fetch":
		fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		conference := fs.String("conference", "", "conference code")
		outDir := fs.String("out", "", "output directory")
		if err := fs.Parse(args[1:]); err != nil {
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
			payload := map[string]any{"conference": conf, "collections": data}
			b, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("encode raw data for %q: %w", *conference, err)
			}
			fmt.Println(string(b))
			return nil
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
		fmt.Printf("Wrote %d raw files to %s\n", count, *outDir)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
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

func printHelp() {
	fmt.Println(`Usage:
  hackertracker conferences
  hackertracker fetch --conference <code> [--out <dir>]

Examples:
  go run ./cmd/hackertracker conferences
  go run ./cmd/hackertracker fetch --conference defcon34 --out ./raw`)
}
