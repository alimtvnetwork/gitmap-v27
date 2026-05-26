// Package cmd — visibilitybulk_test.go
//
// Unit tests for bulk-visibility parsing helpers.
package cmd

import (
	"testing"
)

func TestExtractBaseAndVersionFromArg_URL(t *testing.T) {
	base, ver := extractBaseAndVersionFromArg("https://github.com/alimtvnetwork/gitmap-v23")
	if base != "gitmap" || ver != 23 {
		t.Fatalf("expected (gitmap, 23), got (%s, %d)", base, ver)
	}
}

func TestExtractBaseAndVersionFromArg_Slug(t *testing.T) {
	base, ver := extractBaseAndVersionFromArg("alimtvnetwork/gitmap-v40")
	if base != "gitmap" || ver != 40 {
		t.Fatalf("expected (gitmap, 40), got (%s, %d)", base, ver)
	}
}

func TestExtractBaseAndVersionFromArg_BareNameVersioned(t *testing.T) {
	base, ver := extractBaseAndVersionFromArg("macro-ahk-v10")
	if base != "macro-ahk" || ver != 10 {
		t.Fatalf("expected (macro-ahk, 10), got (%s, %d)", base, ver)
	}
}

func TestExtractBaseAndVersionFromArg_UnversionedDefaultsTo1(t *testing.T) {
	base, ver := extractBaseAndVersionFromArg("gitmap")
	if base != "gitmap" || ver != 1 {
		t.Fatalf("expected (gitmap, 1), got (%s, %d)", base, ver)
	}
}

func TestExtractBaseAndVersionFromArg_UnversionedURL(t *testing.T) {
	base, ver := extractBaseAndVersionFromArg("https://github.com/alimtvnetwork/gitmap")
	if base != "gitmap" || ver != 1 {
		t.Fatalf("expected (gitmap, 1), got (%s, %d)", base, ver)
	}
}

func TestParseBulkRequest_TwoArgValid(t *testing.T) {
	req, ok := parseBulkRequest([]string{"gitmap-v25", "3"})
	if !ok {
		t.Fatal("expected ok=true for valid two-arg request")
	}
	if req.BaseRepo != "gitmap" {
		t.Fatalf("expected BaseRepo=gitmap, got %s", req.BaseRepo)
	}
	if req.StartVer != 25 {
		t.Fatalf("expected StartVer=25, got %d", req.StartVer)
	}
	if req.Count != 3 {
		t.Fatalf("expected Count=3, got %d", req.Count)
	}
}

func TestParseBulkRequest_Empty(t *testing.T) {
	_, ok := parseBulkRequest([]string{})
	if ok {
		t.Fatal("expected ok=false for empty positional")
	}
}

func TestParseBulkRequest_SingleNonInt(t *testing.T) {
	_, ok := parseBulkRequest([]string{"gitmap"})
	if ok {
		t.Fatal("expected ok=false for single non-integer arg")
	}
}

func TestParseBulkRequest_TooManyArgs(t *testing.T) {
	_, ok := parseBulkRequest([]string{"a", "b", "c"})
	if ok {
		t.Fatal("expected ok=false for three positional args")
	}
}