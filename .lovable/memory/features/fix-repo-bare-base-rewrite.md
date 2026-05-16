---
name: fix-repo bare-base rewrite
description: When v1 is a target, fix-repo also rewrites bare `{base}` (pre-versioned remote name) → `{base}-v{current}` with word-boundary guards
type: feature
---

`gitmap fix-repo` (v5.8.0+) extends its rewrite engine so that when v1
is included in the target span, the sweep ALSO substitutes standalone
`{base}` tokens — not just `{base}-v1`. This handles the common case
where the original repo shipped without a `-v1` suffix (e.g.
`alimtvnetwork/img-pdf` instead of `img-pdf-v1`), leaving downstream
references reading the bare name.

**Guard rules** (`isBareBaseBoundary` in `gitmap/cmd/fixrepo_rewrite.go`):
prev byte AND next byte must NOT be a "word char" — defined as ASCII
alnum, `_`, `-`, or `.`. This guarantees:
- `{base}-v2`, `{base}-v10` are skipped (next byte `-`)
- `{base}.js`, `{base}_alt` are skipped (extension / suffix)
- `myimg-pdf` is skipped (prev byte is letter)
- bare `img-pdf` between whitespace / `/` / quotes / line edges is rewritten

The bare-base pass runs ONLY when `n == 1` inside `applyAllTargets`, so
mid-span sweeps that exclude v1 retain the original behavior.

Tests: `gitmap/cmd/fixrepo_rewrite_barebase_test.go`.
