/** @deprecated Use gitmap/constants/constants.go as the single source of truth. */
export const VERSION = "5.64.0" as const;

/** Animation timing defaults (ms). */
export const TERMINAL_INPUT_DELAY = 600;
export const TERMINAL_OUTPUT_DELAY = 120;

/** Watch dashboard refresh interval (seconds). */
export const WATCH_REFRESH_INTERVAL = 30;

/** Status indicator symbols. */
export const STATUS_ICON_DIRTY = "●";
export const STATUS_ICON_CLEAN = "✔";

/** Root-relative path placeholder. */
export const ROOT_RELATIVE_PATH = ".";
export const ROOT_RELATIVE_LABEL = "(root)";

/**
 * Application version — single source of truth FOR THE WEB APP.
 *
 * MUST be kept in sync with the Go binary version declared in
 * `gitmap-v23/constants/constants.go` (`const Version = "..."`). The
 * regression test in `src/test/version-sync.test.ts` reads that file
 * and fails CI if the two drift, so a forgotten bump here will be
 * caught loudly instead of shipping a stale "Current version" badge.
 *
 * Format: `v` + the Go literal (Go uses bare semver, web uses `v` prefix).
 */
export const VERSION = "v5.63.0";
