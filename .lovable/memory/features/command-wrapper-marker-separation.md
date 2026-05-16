---
name: Command Wrapper Marker Separation
description: `gitmap`/`gcd` command-wrapper marker and runtime sentinel are separate from PATH snippet marker/env; fixes false wrapper-active reports. v5.3.0+.
type: feature
---

# Command Wrapper Marker Separation (v5.3.0+)

## Rule

Never use the PATH snippet marker (`# gitmap shell wrapper v2 - managed by ...`)
or `GITMAP_WRAPPER` as proof that the interactive `gitmap` shell function is
installed/active.

The actual command wrapper must use:

- Profile marker: `# gitmap command wrapper v1`
- Runtime sentinel: `GITMAP_COMMAND_WRAPPER=1`

## Root cause fixed

The PATH snippet and command wrapper both used `# gitmap shell wrapper v2`, and
the PATH snippet exported `GITMAP_WRAPPER=1`. That made `gitmap setup` skip
installing the real `function gitmap { ... }` / `gcd` block and made doctor/setup
verification report success even though PowerShell still resolved `gitmap` as the
exe. Result: `gitmap cd <repo>` printed a path but could not change directory.

## Prevention

- `completion.appendCDFunction` must only skip when `CDFuncMarker` is present.
- `isWrapperActive` must check `EnvGitmapCommandWrapper`, not `EnvGitmapWrapper`.
- Keep `GITMAP_WRAPPER` for legacy compatibility only; do not use it for active
  command-wrapper detection.