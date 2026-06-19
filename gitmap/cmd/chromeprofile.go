// Package cmd — chromeprofile.go: entry points for the Chrome profile
// copy/export/import/list pipeline.
//
//	cpc : copy a profile dir (offline, no sign-in tokens)
//	cpe : export profile to a JSON snapshot
//	cpi : import a JSON snapshot back into a profile dir
//	cpl : list profiles discovered under Chrome User Data
//
// Full spec: spec/04-generic-cli/40-chrome-profile-copy.md.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/alimtvnetwork/gitmap-v26/gitmap/constants"
)

// runChromeProfileCopy implements `gitmap chrome-profile-copy`.
func runChromeProfileCopy(args []string) {
	checkHelp(constants.CmdChromeProfileCopy, args)
	if len(args) < 2 {
		fmt.Fprint(os.Stderr, constants.ErrChromeProfileUsageCopy)
		os.Exit(constants.ExitChromeProfileUsage)
	}
	srcProfile, ok := resolveChromeProfile(args[0])
	dstProfile := chromeProfileDestination(args[1])
	if !ok {
		fmt.Fprintf(os.Stderr, constants.ErrChromeProfileSrcMissing, args[0], srcProfile.Path)
		printAvailableChromeProfilesWithDisplay()
		os.Exit(constants.ExitChromeProfileNotFound)
	}
	fmt.Fprint(os.Stderr, constants.MsgChromeProfileSkipChrome)
	fmt.Printf(constants.MsgChromeProfileCopyStart, chromeProfileSummary(srcProfile), chromeProfileSummary(dstProfile), srcProfile.Path, dstProfile.Path)
	start := time.Now()
	files, err := copyChromeProfile(srcProfile.Path, dstProfile.Path)
	if err != nil {
		printChromeProfileCopyError(srcProfile, dstProfile, err)
		os.Exit(constants.ExitChromeProfileCopyFailed)
	}
	fmt.Printf(constants.MsgChromeProfileCopyDone, files, time.Since(start).Round(time.Millisecond))
	rec := emitChromeSnapshots(dstProfile.Path, args[1])
	persistChromeProfile(args[1], dstProfile.Path, rec)
}

// emitChromeSnapshots writes the JSON + CSV companions for a profile
// and prints both paths in a consistent Artifacts block. Used by cpc
// and cpe so the output is identical and copy-paste friendly.
func emitChromeSnapshots(srcPath, name string) chromeExportRecord {
	jsonPath := defaultChromeExportPath(name)
	jsonBytes, err := writeChromeExport(srcPath, name, jsonPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrChromeProfileExportFail, err)
		return chromeExportRecord{}
	}
	csvPath := jsonPath[:len(jsonPath)-len(constants.ExtJSON)] + constants.ExtCSV
	csvBytes, err := writeChromeExportCSV(srcPath, name, csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrChromeProfileExportFail, err)
		csvPath = ""
	}
	rec := chromeExportRecord{JSONPath: jsonPath, JSONSize: jsonBytes, CSVPath: csvPath, CSVSize: csvBytes}
	printChromeArtifacts(rec)
	return rec
}

// printChromeArtifacts prints the canonical Artifacts: block. Always
// emits both rows so callers can grep `json:`/`csv:` deterministically.
func printChromeArtifacts(rec chromeExportRecord) {
	fmt.Print(constants.MsgChromeProfileArtifactsHd)
	fmt.Printf(constants.MsgChromeProfileArtifactRow, "json:", artifactValue(rec.JSONPath))
	fmt.Printf(constants.MsgChromeProfileArtifactRow, "csv:", artifactValue(rec.CSVPath))
}

func artifactValue(path string) string {
	if path == "" {
		return constants.MsgChromeProfileArtifactNA
	}
	return path
}

// runChromeProfileExport implements `gitmap chrome-profile-export`.
func runChromeProfileExport(args []string) {
	checkHelp(constants.CmdChromeProfileExport, args)
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, constants.ErrChromeProfileUsageExport)
		os.Exit(constants.ExitChromeProfileUsage)
	}
	name := args[0]
	outPath := defaultChromeExportPath(name)
	if len(args) >= 2 {
		outPath = args[1]
	}
	srcPath, ok := resolveChromeProfileDir(name)
	if !ok {
		fmt.Fprintf(os.Stderr, constants.ErrChromeProfileSrcMissing, name, srcPath)
		printAvailableChromeProfilesWithDisplay()
		os.Exit(constants.ExitChromeProfileNotFound)
	}
	jsonBytes, err := writeChromeExport(srcPath, name, outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrChromeProfileExportFail, err)
		os.Exit(constants.ExitChromeProfileCopyFailed)
	}
	csvPath := outPath
	if ext := constants.ExtJSON; len(csvPath) > len(ext) && csvPath[len(csvPath)-len(ext):] == ext {
		csvPath = csvPath[:len(csvPath)-len(ext)] + constants.ExtCSV
	} else {
		csvPath += constants.ExtCSV
	}
	csvBytes, csvErr := writeChromeExportCSV(srcPath, name, csvPath)
	if csvErr != nil {
		fmt.Fprintf(os.Stderr, constants.ErrChromeProfileExportFail, csvErr)
		csvPath = ""
	}
	rec := chromeExportRecord{
		JSONPath: outPath, JSONSize: jsonBytes,
		CSVPath: csvPath, CSVSize: csvBytes,
	}
	printChromeArtifacts(rec)
	persistChromeProfile(name, srcPath, rec)
}

// runChromeProfileImport implements `gitmap chrome-profile-import`.
// Accepts both .json (full snapshot) and .csv (lossy: extension IDs +
// known preferences only — bookmarks omitted).
func runChromeProfileImport(args []string) {
	checkHelp(constants.CmdChromeProfileImport, args)
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, constants.ErrChromeProfileUsageImport)
		os.Exit(constants.ExitChromeProfileUsage)
	}
	srcFile := args[0]
	exp, err := loadChromeImport(srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrChromeProfileImportFail, err)
		os.Exit(constants.ExitChromeProfileCopyFailed)
	}
	name := exp.Name
	if len(args) >= 2 {
		name = args[1]
	}
	dstPath := chromeProfilePath(name)
	if err := applyChromeExport(exp, dstPath); err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrChromeProfileImportFail, err)
		os.Exit(constants.ExitChromeProfileCopyFailed)
	}
	fmt.Printf(constants.MsgChromeProfileImportOk, srcFile, name)
}

// runChromeProfileList implements `gitmap chrome-profile-list`.
func runChromeProfileList(args []string) {
	checkHelp(constants.CmdChromeProfileList, args)
	root := chromeUserDataDir()
	entries := chromeProfileEntries()
	if len(entries) == 0 {
		fmt.Printf(constants.MsgChromeProfileListEmpty, root)
		listChromeProfilesFromDB()
		return
	}
	fmt.Printf(constants.MsgChromeProfileListHdr, root)
	for _, e := range entries {
		if e.DisplayName != "" {
			fmt.Printf("  - %s  (display: %q)\n", e.Dir, e.DisplayName)
			continue
		}
		fmt.Printf("  - %s\n", e.Dir)
	}
	listChromeProfilesFromDB()
}

// defaultChromeExportPath builds the default JSON output location
// under .gitmap/chrome/<name>.json (cwd-relative).
func defaultChromeExportPath(name string) string {
	return filepath.Join(constants.GitMapDir, "chrome", name+constants.ExtJSON)
}



// readChromeExport loads a JSON export file from disk.
func readChromeExport(path string) (*chromeExport, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // user-supplied path
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var exp chromeExport
	if err := json.Unmarshal(raw, &exp); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &exp, nil
}

// copyChromeProfile copies the curated subset of entries from src to
// dst. Missing entries are skipped silently — Chrome regenerates them.
func copyChromeProfile(src, dst string) (int, error) {
	if err := os.MkdirAll(dst, constants.DirPermission); err != nil {
		return 0, newChromeProfileCopyError(src, dst, constants.ChromeProfileCopyOpMkdir, err)
	}
	total := 0
	for _, name := range constants.ChromeProfileCopyEntries {
		n, err := copyEntry(filepath.Join(src, name), filepath.Join(dst, name))
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

type chromeProfileCopyError struct {
	Source string
	Target string
	Op     string
	Err    error
}

func (e *chromeProfileCopyError) Error() string {
	return fmt.Sprintf("%s %s -> %s: %v", e.Op, e.Source, e.Target, e.Err)
}

func (e *chromeProfileCopyError) Unwrap() error { return e.Err }

func newChromeProfileCopyError(src, dst, op string, err error) error {
	return &chromeProfileCopyError{Source: src, Target: dst, Op: op, Err: err}
}

// copyEntry copies a single file or directory tree. Returns file count.
func copyEntry(src, dst string) (int, error) {
	info, err := os.Stat(src)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return 0, newChromeProfileCopyError(src, dst, constants.ChromeProfileCopyOpStat, err)
		}
		return 0, nil
	}
	if !info.IsDir() {
		return 1, chromeProfileCopyFile(src, dst)
	}
	return copyDir(src, dst)
}

// chromeProfileCopyFile copies a single file from src to dst preserving mode.
func chromeProfileCopyFile(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec // curated entry list
	if err != nil {
		if isChromeVolatileLockFile(src) {
			warnChromeProfileLockSkip(src, dst, err)
			return nil
		}
		return newChromeProfileCopyError(src, dst, constants.ChromeProfileCopyOpRead, err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), constants.DirPermission); err != nil {
		return newChromeProfileCopyError(src, dst, constants.ChromeProfileCopyOpMkdir, err)
	}
	out, err := os.Create(dst) //nolint:gosec // curated entry list
	if err != nil {
		return newChromeProfileCopyError(src, dst, constants.ChromeProfileCopyOpWrite, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return newChromeProfileCopyError(src, dst, constants.ChromeProfileCopyOpWrite, err)
	}
	return nil
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) (int, error) {
	if err := os.MkdirAll(dst, constants.DirPermission); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, e := range entries {
		n, err := copyEntry(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()))
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}
