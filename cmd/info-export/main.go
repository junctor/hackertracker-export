package main

import (
	"context"
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

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("info-export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	conference := fs.String("conference", "", "conference code")
	conferenceShort := fs.String("c", "", "conference code")
	outDir := fs.String("out", "", "output directory")
	outShort := fs.String("o", "", "output directory")
	help := fs.Bool("help", false, "show help")
	helpShort := fs.Bool("h", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help || *helpShort {
		printHelp()
		return nil
	}
	confCode := firstNonEmpty(*conference, *conferenceShort)
	if confCode == "" && fs.NArg() > 0 {
		confCode = fs.Arg(0)
	}
	if confCode == "" {
		printHelp()
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
		return err
	}
	artifacts, err := transform.Build(conf, data, transform.BuildOptions{
		SchemaVersion:  2,
		BuildTimestamp: time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	written, err := export.WriteArtifacts(out, artifacts)
	if err != nil {
		return err
	}
	fmt.Printf("Exported %s -> %s\n", conf.Code, out)
	fmt.Printf("Wrote %d files\n", len(written))
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func printHelp() {
	fmt.Println(`Usage:
  info-export --conference <code> --out <dir>

Options:
  --conference, -c <code>  Conference code
  --out, -o <dir>          Output directory

Examples:
  go run ./cmd/info-export --conference defcon34 --out ./public/defcon34/data`)
}
