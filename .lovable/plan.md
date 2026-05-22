## Goal

Make `gitmap` / `gitmap help` look professional in every terminal (Windows PowerShell, Windows Terminal, macOS Terminal, iTerm2, Linux gnome-terminal), fix the broken/gibberish emoji rendering on PowerShell, and make it easy for users to discover and drill into commands. Each step below is a self-contained unit of work that will be executed one at a time. After each step I will list what is done and what remains. The final step bumps the minor version, writes the changelog, and pins the new version to the root README.

---

### Step 1 — Universal-safe glyph system (fix PowerShell "gibberish" emojis)

**Problem:** Emojis like 📋 ✅ 🎉 render as `≡ƒôï` / `Γ£à` on default PowerShell (cp1252) and on terminals that lack an emoji font, even though `initConsole` switches to UTF-8 and enables VT processing. Two failure modes:
1. Old `powershell.exe` (5.1) where the input/output encoding is reset by the host before we can intercept.
2. Fonts without color-emoji glyphs (Consolas, Lucida Console) — rendered as tofu/boxes even when UTF-8 is correct.

**Plan:**
- Introduce a glyph layer in `gitmap/constants/constants_glyphs.go` with two sets:
  - **Rich set** — current emoji (📋 ✅ ⚠ 🎉 🔑 📁 🏷 ✓ →).
  - **Safe set** — universally-rendered ASCII + BMP fallbacks (`[OK]`, `[!]`, `[i]`, `->`, `*`, `+`, `x`, `v`).
- Add a `--glyphs <auto|rich|safe>` global flag + `GITMAP_GLYPHS` env var (mirrors the existing `--theme` pattern).
- Auto-detection: on Windows, treat `safe` as default when `$env:WT_SESSION` is empty AND host is `ConsoleHost` (legacy powershell.exe); use `rich` in Windows Terminal, VS Code, iTerm2, all *nix TTYs.
- Replace every literal emoji in Msg* constants and runtime `fmt.Print` calls with `glyph.OK`, `glyph.Warn`, `glyph.Info`, `glyph.Arrow`, etc. resolved at print time.
- Harden `initConsole` to also set `Console.InputEncoding`/`OutputEncoding` analog via `_setmode` for stdout where applicable, and document that PowerShell 5.1 users should run `gitmap` from Windows Terminal for rich glyphs.

**Deliverable:** No more mojibake anywhere. Users on legacy PowerShell see clean ASCII; users on modern terminals keep the polished look.

---

### Step 2 — Redesign `gitmap` / `gitmap help` root output

**Problem:** The current compact root listing is a wall of group headers + dense one-liners. Hard to scan, no visual hierarchy, no clear "what should I run first?" path.

**Plan:**
- Rebuild `printCompactAll` in `gitmap/cmd/rootusagecompact.go` into a structured, columnar layout:
  - Top banner: name, version, tagline, source repo, install dir (already partially in `rootusagefooter.go` — promote to header).
  - **Quick Start** strip (4 most common commands: `scan`, `clone`, `pull-release`, `setup`) with one-line purpose each.
  - Two-column **group → commands** grid, fixed width, aligned, grouped by user intent (Get started / Daily / Release / Power tools / Diagnostics) instead of the current 17 flat groups.
  - Each command line: `name (alias)` padded to col 22, then one-sentence purpose, dim-colored.
  - Footer: `gitmap help <cmd>` hint, `gitmap help --filter <word>` hint, build info.
- Adapt width to terminal columns (read via `golang.org/x/term`) with a sane 80-col fallback.
- Apply theme palette already in place (`--theme`) so colors degrade in `standard` / `mono`.

**Deliverable:** A root help screen that looks like a polished CLI (think `gh`, `cargo`, `rg --help`).

---

### Step 3 — Add a filter / search option

**Problem:** No way to search commands by keyword. With 80+ subcommands users can't find what they need.

**Plan:**
- Add `gitmap help --filter <query>` (alias `-f`) and bare-positional `gitmap help <query>`.
- Match against: command name, aliases, group key, and the one-line description (already present per command).
- Output: same column layout as Step 2 but filtered to matches, with the matched substring highlighted via the theme accent color.
- If exactly one match → print that command's full help (delegates to existing `helptext.Print`).
- If zero matches → suggest the 3 closest via Levenshtein distance.
- Document in `gitmap/helptext/help.md` (new file) and link from root footer.

**Deliverable:** `gitmap help release`, `gitmap help -f ssh`, `gitmap help clone` all do the right thing instantly.

---

### Step 4 — Per-command detailed help: audit + fill gaps

**Problem:** User wants every command to have a richer "if I go into that, that would have an explanation of how that command works". Many help files exist but several are stubs or missing (e.g. recently added `push`, `pull`, `prc`, `undo`, `ssh copy/view/create`, `install gitmap-oneliner`, `reinstall`).

**Plan:**
- Audit `gitmap/helptext/*.md` against the canonical command list in `constants_cli.go` and the registry tests; produce a coverage matrix.
- For every missing or thin file, author one following the spec in `spec/04-generic-cli/09-help-system.md` (≤120 lines, 2–3 examples with realistic output, Prerequisites, See Also).
- Standardize section order across all files (Usage → Aliases → Flags table → Prerequisites → Examples → Common pitfalls → See Also) so the pretty renderer produces a uniform look.
- Add a CI check (or unit test) that fails when a registered command lacks a help file.

**Deliverable:** Every `gitmap <cmd> --help` returns a complete, well-formatted page.

---

### Step 5 — Version bump, changelog, README pin

**Plan (executed only after Steps 1–4 are verified):**
- Bump minor: `v5.41.0 → v5.42.0` in `gitmap/constants/constants.go`, `src/constants/index.ts`.
- Add `CHANGELOG.md` and `src/data/changelog.ts` entry summarizing: universal glyphs, redesigned root help, `help --filter`, full per-command help coverage.
- Update all 15+ version pins in `README.md` (the version matrix + install URL examples).
- Run `src/test/version-sync.test.ts` and the Go test suite to confirm sync.
- Suggest publish.

---

### Open questions (please confirm or I'll proceed with the defaults shown)

1. **Glyph default on legacy PowerShell 5.1:** default to `safe` (ASCII) unless user opts into `--glyphs rich`? *(Default: yes)*
2. **Grouping in the new root help:** collapse the 17 current groups into ~5 intent-based buckets? *(Default: yes — keep old keys as aliases for `--filter`)*
3. **`help` as a first-class subcommand vs. flag:** keep both `gitmap help` and `gitmap --help`, with `help` getting the new filter/search powers? *(Default: yes)*
4. **Final version target:** `v5.42.0`? *(Default: yes)*

Reply "go" (or with any tweaks) and I'll execute Step 1 next, then report status + remaining steps.
