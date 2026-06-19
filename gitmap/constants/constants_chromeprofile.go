// Package constants — Chrome profile copy/export/import command IDs,
// help text, messages, and exit codes for `gitmap chrome-profile-copy`
// and friends. Spec: spec/04-generic-cli/40-chrome-profile-copy.md.
package constants

// gitmap:cmd top-level
// Chrome profile command IDs and short aliases.
const (
	CmdChromeProfileCopy        = "chrome-profile-copy"
	CmdChromeProfileCopyAlias   = "cpc"
	CmdChromeProfileExport      = "chrome-profile-export"
	CmdChromeProfileExportAlias = "cpe"
	CmdChromeProfileImport      = "chrome-profile-import"
	CmdChromeProfileImportAlias = "cpi"
	CmdChromeProfileList        = "chrome-profile-list"
	CmdChromeProfileListAlias   = "cpl"
)

// Chrome profile help-line entries surfaced by `gitmap help`.
const (
	HelpChromeProfileCopy   = "  chrome-profile-copy (cpc) <src> <dst>   Copy a Chrome profile (bookmarks, extensions, prefs, flags) into an offline profile"
	HelpChromeProfileExport = "  chrome-profile-export (cpe) <name> [out] Export profile to JSON (default: .gitmap/chrome/<name>.json)"
	HelpChromeProfileImport = "  chrome-profile-import (cpi) <file> [name] Import a Chrome profile from a JSON export"
	HelpChromeProfileList   = "  chrome-profile-list (cpl)               List Chrome profiles known to gitmap"
)

// Chrome profile messages and errors.
const (
	MsgChromeProfileCopyStart  = "chrome-profile-copy: %s → %s\n"
	MsgChromeProfileCopyDone   = "chrome-profile-copy: done (%d files, %s)\n"
	MsgChromeProfileExportOk   = "chrome-profile-export: wrote %s (%d bytes)\n"
	MsgChromeProfileExportCSV  = "chrome-profile-export: csv  %s (%d bytes)\n"
	MsgChromeProfileDBSynced   = "chrome-profile: db synced (%s)\n"
	MsgChromeProfileDBWarn     = "  ⚠ chrome-profile: db sync failed: %v\n"
	MsgChromeProfileImportOk   = "chrome-profile-import: imported %s into profile %q\n"
	MsgChromeProfileListEmpty  = "chrome-profile-list: no profiles found at %s\n"
	MsgChromeProfileListHdr    = "Chrome profiles (%s):\n"
	MsgChromeProfileListDBHdr  = "Tracked in gitmap DB:\n"
	MsgChromeProfileListDBRow  = "  - %-30s  exports=%d  last=%s\n"
	MsgChromeProfileSkipChrome = "  Hint: close Chrome before copying — open sessions may corrupt the destination.\n"

	ErrChromeProfileUsageCopy   = "chrome-profile-copy: ERROR <src> and <dst> are required\n  usage: gitmap chrome-profile-copy <src-profile> <dst-profile>\n"
	ErrChromeProfileUsageExport = "chrome-profile-export: ERROR <name> is required\n  usage: gitmap chrome-profile-export <name> [out.json]\n"
	ErrChromeProfileUsageImport = "chrome-profile-import: ERROR <file> is required\n  usage: gitmap chrome-profile-import <file.json> [dst-profile]\n"
	ErrChromeProfileSrcMissing  = "chrome-profile-copy: ERROR source profile %q not found at %s\n"
	ErrChromeProfileCopyFailed  = "chrome-profile-copy: ERROR copy failed: %v\n"
	ErrChromeProfileExportFail  = "chrome-profile-export: ERROR %v\n"
	ErrChromeProfileImportFail  = "chrome-profile-import: ERROR %v\n"
)

// Chrome User Data subpaths copied by cpc. Excluded by design:
// Cookies, Login Data, History, Cache, GPUCache, sync tokens.
var ChromeProfileCopyEntries = []string{
	"Bookmarks",
	"Favicons",
	"Preferences",
	"Secure Preferences",
	"Extensions",
	"Local Extension Settings",
	"Extension Rules",
	"Extension State",
	"Sync Extension Settings",
	"Web Data",
	"Shortcuts",
	"TransportSecurity",
}

// Chrome User Data top-level files (siblings of profile dirs).
var ChromeUserDataTopLevel = []string{
	"Local State",
}

// Chrome profile exit codes.
const (
	ExitChromeProfileOk         = 0
	ExitChromeProfileUsage      = 6
	ExitChromeProfileNotFound   = 7
	ExitChromeProfileCopyFailed = 10
)
