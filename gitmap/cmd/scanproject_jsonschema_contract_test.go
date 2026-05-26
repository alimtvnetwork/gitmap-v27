package cmd

// JSON-schema contract for `gitmap scan-project` per-type files.
//
// `scan-project` writes 5 sibling JSON files (`go-projects.json`,
// `node-projects.json`, `react-projects.json`, `cpp-projects.json`,
// `csharp-projects.json`) — each is a JSON array of detection
// records produced by `buildJSONRecords`. Top-level record keys are
// PascalCase (`Project`, `GoMeta`, `Csharp`) because
// `detector.DetectionResult` has no `json:` tags; this is the
// contractual on-the-wire shape since v1.
//
// These tests pin:
//   1. The 5 file names emitted by `projectTypeJSONMap` exactly
//      match the registry copy.
//   2. Every key produced by `buildJSONRecords` for both the
//      bare-record path (Node/React/Cpp) and the metadata-wrapped
//      path (Go/Csharp) is declared in the schema's
//      `items.properties` map.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/alimtvnetwork/gitmap-v23/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v23/gitmap/detector"
	"github.com/alimtvnetwork/gitmap-v23/gitmap/model"
)

const scanProjectSchemaFilename = "scan-project.schema.json"

// TestScanProject_FileMapMatchesRegistry locks the set of emitted
// filenames against the schema-registry copy so silent additions
// or renames break CI.
func TestScanProject_FileMapMatchesRegistry(t *testing.T) {
	raw, err := json.Marshal(struct {
		Files []string `json:"files"`
	}{Files: []string{
		constants.JSONFileGoProjects,
		constants.JSONFileNodeProjects,
		constants.JSONFileReactProjects,
		constants.JSONFileCppProjects,
		constants.JSONFileCsharpProjects,
	}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	regBytes := loadRegistryRaw(t, "scan-project.v1.json")
	var reg struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(regBytes, &reg); err != nil {
		t.Fatalf("parse registry: %v", err)
	}

	var got struct {
		Files []string `json:"files"`
	}
	_ = json.Unmarshal(raw, &got)
	if !equalStringSlices(got.Files, reg.Files) {
		t.Errorf("scan-project files drift:\n got=%v\nwant=%v", got.Files, reg.Files)
	}

	// Defensive: the map MUST contain every advertised filename.
	for _, want := range reg.Files {
		if !containsString(got.Files, want) {
			t.Errorf("registry advertises %q but projectTypeJSONMap omits it", want)
		}
	}
}

// TestScanProject_RecordKeysSubsetOfSchema runs the live record
// builder for both DetectionResult shape variants and asserts every
// emitted key is declared in the schema's items.properties map.
func TestScanProject_RecordKeysSubsetOfSchema(t *testing.T) {
	root := loadSchemaFile(t, scanProjectSchemaFilename)
	items, _ := root["items"].(map[string]any)
	props, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatalf("items.properties missing")
	}

	results := []detector.DetectionResult{
		{Project: model.DetectedProject{ID: 1, ProjectType: constants.ProjectKeyNode, ProjectName: "demo-node"}},
		{
			Project: model.DetectedProject{ID: 2, ProjectType: constants.ProjectKeyGo, ProjectName: "demo-go"},
			GoMeta:  &model.GoProjectMetadata{ID: 1, ModuleName: "example.com/demo"},
		},
		{
			Project: model.DetectedProject{ID: 3, ProjectType: constants.ProjectKeyCsharp, ProjectName: "demo-cs"},
			Csharp:  &model.CsharpProjectMetadata{ID: 1, SlnName: "Demo.sln"},
		},
	}

	records := buildJSONRecords(results)
	raw, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}

	keysPerRecord := readEveryObjectKeys(t, raw)
	if len(keysPerRecord) != len(results) {
		t.Fatalf("expected %d records, got %d", len(results), len(keysPerRecord))
	}
	for i, keys := range keysPerRecord {
		for _, key := range keys {
			if _, allowed := props[key]; !allowed {
				t.Errorf("record[%d]: encoder emitted %q not declared in scan-project schema", i, key)
			}
		}
	}
}

// containsString is a tiny local helper kept out of the shared
// helpers file to avoid bloating its surface area.
func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}

	return false
}
