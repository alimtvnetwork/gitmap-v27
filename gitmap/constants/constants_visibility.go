// Package constants — visibility command IDs, flags, messages, and
// exit codes for `gitmap make-public` / `gitmap make-private`.
//
// The two commands are thin wrappers around the host platform's CLI
// (`gh` for GitHub, `glab` for GitLab). They:
//
//  1. Resolve provider + owner/repo from `git remote get-url origin`.
//  2. Read the current visibility via the provider CLI.
//  3. Skip if already in the target state (idempotent).
//  4. Prompt the user when going private → public (skip with --yes).
//  5. Apply, then verify the change took effect.
//
// Spec parity: spec-authoring/23-visibility-change/01-spec.md
// (PowerShell reference: visibility-change.ps1).
package constants

// Visibility command IDs live in constants_cli.go (CmdMakePublic /
// CmdMakePrivate) per the project-wide rule that all CLI tokens are
// centralized there. This file owns everything else (target tokens,
// flags, messages, exit codes).

// Visibility target tokens — what the provider CLI expects, what the
// user can type for the (optional) explicit-target form, and what we
// store/print internally.
const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"

	VisShortPub = "pub"
	VisShortPri = "pri"
)

// Visibility flags. --yes skips the private→public confirmation;
// --dry-run prints what would change without invoking the provider
// CLI; --verbose echoes each shell command before running it.
const (
	FlagVisYes     = "yes"
	FlagVisYesAlt  = "y"
	FlagVisDryRun  = "dry-run"
	FlagVisVerbose = "verbose"

	FlagDescVisYes     = "Skip the private→public confirmation prompt (no-op for public→private)."
	FlagDescVisDryRun  = "Print the provider CLI command that would run; do not invoke it."
	FlagDescVisVerbose = "Echo every shell command to stderr before running it."
)

// Provider tokens — match what we detect from the origin URL host.
const (
	ProviderGitHub = "github"
	ProviderGitLab = "gitlab"

	HostGitHub = "github.com"
	HostGitLab = "gitlab.com"

	CLIGitHub = "gh"
	CLIGitLab = "glab"
)

// Visibility help-line entries surfaced by `gitmap help` (Utilities).
const (
	HelpMakePublic  = "  make-public         Make current repo public on GitHub/GitLab (gh/glab required)"
	HelpMakePrivate = "  make-private        Make current repo private on GitHub/GitLab (gh/glab required)"
)

// Visibility user-facing messages.
const (
	MsgVisAlreadyFmt  = "visibility: already %s on %s\n"
	MsgVisChangedFmt  = "visibility: %s → %s on %s (%s)\n"
	MsgVisDryRunFmt   = "[dry-run] visibility: %s → %s on %s (%s)\n"
	MsgVisConfirmFmt  = "Make %s PUBLIC on %s? Type 'yes' to confirm: "
	MsgVisVerboseExec = "+ %s %s\n"
	MsgVisVerifyOK    = "  ✓ verified: visibility is now %s\n"
)

// Visibility error messages.
const (
	ErrVisNotInRepo       = "visibility: not a git repository\n"
	ErrVisNoOrigin        = "visibility: no `origin` remote configured\n"
	ErrVisBadProviderFmt  = "visibility: unsupported host in %q (only github.com / gitlab.com are supported)\n"
	ErrVisBadSlugFmt      = "visibility: cannot parse owner/repo from %q\n"
	ErrVisCLIMissingFmt   = "visibility: %q not found on PATH (install: https://cli.github.com or https://gitlab.com/gitlab-org/cli)\n"
	ErrVisReadCurrentFmt  = "visibility: cannot read current visibility (auth via `%s auth login`?): %v\n"
	ErrVisConfirmRequired = "visibility: confirmation required (re-run with --yes for non-interactive use)\n"
	ErrVisApplyFailedFmt  = "visibility: apply failed: %v\n%s"
	ErrVisVerifyFailedFmt = "visibility: verification failed — current is %q, expected %q\n"
)

// Visibility exit codes (mirrored from visibility-change.ps1 so wrappers
// and CI can branch on the same numbers).
const (
	ExitVisOK           = 0
	ExitVisNotARepo     = 2
	ExitVisNoOrigin     = 3
	ExitVisBadProvider  = 4
	ExitVisAuthFailed   = 5
	ExitVisBadFlag      = 6
	ExitVisConfirmReq   = 7
	ExitVisVerifyFailed = 8
)

// Spec 113 — bulk visibility + cfrp prior-version privatize messages.
const (
	MsgVisBulkHeaderFmt    = "visibility: %s × %d versions of %s on %s\n"
	MsgVisBulkItemFmt      = "  [%d/%d] %s … "
	MsgVisBulkSkipFmt      = "already %s\n"
	MsgVisBulkOKFmt        = "%s → %s\n"
	MsgVisBulkFailFmt      = "FAILED (%v)\n"
	MsgVisBulkDryFmt       = "[dry-run] %s → %s (slug=%s)\n"
	MsgCFRPPriorHeaderFmt  = "\ncfrp: scanning prior versions of %s (≤%d back)…\n"
	MsgCFRPPriorFoundFmt   = "cfrp: %d prior version(s) currently public: %s\n"
	MsgCFRPPriorPromptFmt  = "Privatize all %d prior version(s)? [y/N]: "
	MsgCFRPPriorNoneFound  = "cfrp: no prior public versions found.\n"
	MsgCFRPPriorSkipped    = "cfrp: leaving prior versions unchanged.\n"
	ErrVisBulkBadCountFmt  = "visibility: count must be a positive integer, got %q\n"
	ErrVisBulkRepoParseFmt = "visibility: cannot parse repo identity from %q\n"
)

// CFRPPriorMaxLookback caps the prior-version probe at v(N-15)..v(N-1)
// — far enough for realistic release histories without abusing the API.
const CFRPPriorMaxLookback = 15

// Spec 116 — bulk wildcard visibility (make-all-public / make-all-private
// / MAPUB / MAPRI). Owner-only resolver + repo-list pagination cap.
const (
	ProviderUnknownReason   = "unknown"
	MsgMakeAllNotImpl       = "make-all-*: handler not yet wired (spec/01-app/116)\n"
	ErrMakeAllResolveFmt    = "make-all-*: cannot resolve owner: %v\n"
	ErrMakeAllMissingArgFmt = "make-all-*: usage: %s <target> <patterns> [-Y|--yes]\n"

	// OwnerRepoListLimit caps `gh/glab repo list --limit`. Owners with
	// more than this many repos will hit a WARNING (see plan step 26).
	// 1000 matches `gh`'s own documented max page; `glab` accepts the
	// same -P value without paginating internally.
	OwnerRepoListLimit      = 1000
	WarnOwnerRepoListCapFmt = "make-all-*: WARNING — owner %[2]s returned %[1]d repos (the --limit cap). Repos beyond the cap were NOT enumerated; narrow the patterns or raise the limit.\n"
)



