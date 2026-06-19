package cli

import (
	"io"
	"reflect"
	"testing"
)

func TestParseInfoExportOptionsSupportsConferenceFlagWithPositionals(t *testing.T) {
	opts, err := parseInfoExportOptions([]string{
		"--conference", "DCSG2026",
		"DEFCON34",
		"DEFCON33",
		"--out", "./public",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	wantCodes := []string{"DCSG2026", "DEFCON34", "DEFCON33"}
	if !reflect.DeepEqual(opts.conferenceCodes, wantCodes) {
		t.Fatalf("conferenceCodes = %#v, want %#v", opts.conferenceCodes, wantCodes)
	}
	if opts.outDir != "./public" {
		t.Fatalf("outDir = %q, want ./public", opts.outDir)
	}
}

func TestParseInfoExportOptionsSupportsRepeatedConferenceFlags(t *testing.T) {
	opts, err := parseInfoExportOptions([]string{
		"-c", "DCSG2026",
		"--conference=DEFCON34",
		"-c=DEFCON33",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"DCSG2026", "DEFCON34", "DEFCON33"}
	if !reflect.DeepEqual(opts.conferenceCodes, want) {
		t.Fatalf("conferenceCodes = %#v, want %#v", opts.conferenceCodes, want)
	}
}

func TestParseInfoExportOptionsDeduplicatesConferenceCodes(t *testing.T) {
	opts, err := parseInfoExportOptions([]string{"DEFCON34", "-c", "DEFCON34", "DEFCON33"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"DEFCON34", "DEFCON33"}
	if !reflect.DeepEqual(opts.conferenceCodes, want) {
		t.Fatalf("conferenceCodes = %#v, want %#v", opts.conferenceCodes, want)
	}
}

func TestInfoExportOutputDir(t *testing.T) {
	tests := []struct {
		name     string
		outRoot  string
		confCode string
		multiple bool
		want     string
	}{
		{name: "default", confCode: "DEFCON34", want: "out/ht/defcon34"},
		{name: "single explicit", outRoot: "./public/defcon34/data", confCode: "DEFCON34", want: "./public/defcon34/data"},
		{name: "multiple explicit", outRoot: "./public", confCode: "DEFCON34", multiple: true, want: "public/defcon34"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := infoExportOutputDir(tt.outRoot, tt.confCode, tt.multiple); got != tt.want {
				t.Fatalf("infoExportOutputDir() = %q, want %q", got, tt.want)
			}
		})
	}
}
