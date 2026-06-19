# gitmap chrome-profile-copy

Copy a Chrome profile directory (bookmarks, extensions, prefs, flags)
into an offline-safe destination profile under Chrome's User Data root.
Sign-in tokens, cookies, history and caches are deliberately excluded.

## Alias

`cpc`

## Usage

    gitmap chrome-profile-copy <src-profile> <dst-profile>
    gitmap cpc <src-profile> <dst-profile>

`<src-profile>` and `<dst-profile>` are profile **names** (`Default`,
`Profile 1`, `Work`, ...), not absolute paths. Both are resolved
relative to Chrome's User Data root for the current OS.

## What it copies

| Included                                         | Excluded (by design)             |
|--------------------------------------------------|----------------------------------|
| Bookmarks, Favicons                              | Cookies, Login Data              |
| Preferences, Secure Preferences                  | History, Cache, GPUCache         |
| Extensions + Local/Rules/State extension data    | Sync tokens / OAuth credentials  |
| Web Data, Shortcuts, TransportSecurity           |                                  |

## Prerequisites

- **Close Chrome first** — open sessions may corrupt the destination.
- The source profile must exist under the User Data root. When it
  doesn't, the not-found error lists every real profile so you can
  pick the correct spelling (v6.34.0+).

## Examples

### Clone "Default" into a fresh "Backup" profile

    $ gitmap cpc Default Backup
    chrome-profile-copy: C:\Users\me\AppData\Local\Google\Chrome\User Data\Default → C:\Users\me\AppData\Local\Google\Chrome\User Data\Backup
    chrome-profile-copy: done (143 files, 412ms)
    Artifacts:
      json  C:\dev\.gitmap\chrome\Backup.json
      csv   C:\dev\.gitmap\chrome\Backup.csv
    chrome-profile: db synced (Backup)

### Typo handling

    $ gitmap cpc Defaultt Backup
    chrome-profile-copy: ERROR source profile "Defaultt" not found at ...\Defaultt
    available profiles under C:\Users\me\AppData\Local\Google\Chrome\User Data:
      Default
      Profile 1
      Profile 2

## Exit codes

| Code | Meaning                          |
|------|----------------------------------|
| 0    | Copy succeeded                   |
| 6    | Usage error (missing args)       |
| 7    | Source profile not found         |
| 10   | Copy failed mid-flight           |

## See also

- [chrome-profile-export](chrome-profile-export.md) — JSON/CSV snapshot
- [chrome-profile-import](chrome-profile-import.md) — restore a snapshot
- [chrome-profile-list](chrome-profile-list.md) — list known profiles
- [chrome-profile-delete](chrome-profile-delete.md) — drop a tracked profile
- Spec: `spec/04-generic-cli/40-chrome-profile-copy.md`
