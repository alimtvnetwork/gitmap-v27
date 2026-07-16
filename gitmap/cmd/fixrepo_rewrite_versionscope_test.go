package cmd

// v5.39.0: regression matrix locking down the bare-base rewrite scope
// across current versions v1..v4. Bare `{base}` substitution must ONLY
// fire on the v1→v2 transition (current==2 with v1 in targets). Every
// other (current, targets) shape must preserve standalone `{base}`.

import "testing"

func TestApplyAllTargets_VersionScopeMatrix(t *testing.T) {
	const base = "gitmap"
	type tc struct {
		name    string
		current int
		targets []int
		in      string
		want    string
	}
	// Distractor tokens deliberately use a non-`gitmap` base
	// (`otherpkg-vN`) so the fix-repo rewriter — which only touches
	// `{base}-vN` tokens where base == this repo's name — cannot
	// silently rewrite these literals on a future version bump.
	// (See mem://constraints fix-repo digit-capture rule + the v6.2.x
	// regression where every `gitmap-vN<25` here got smashed to v25.)
	cases := []tc{
		{
			// current=1 is a no-op floor: nothing to bump to.
			name:    "v1_no_rewrite",
			current: 1,
			targets: []int{},
			in:      "gitmap and otherpkg-v9 stay put",
			want:    "gitmap and otherpkg-v9 stay put",
		},
		{
			// v1→v2: the ONLY case where bare base is rewritten.
			name:    "v2_bare_base_rewritten",
			current: 2,
			targets: []int{1},
			in:      "url=https://github.com/x/gitmap plus otherpkg-v9 token",
			want:    "url=https://github.com/x/gitmap-v2 plus otherpkg-v9 token",
		},
		{
			// v3: bare base preserved even with v1 in targets.
			name:    "v3_bare_base_preserved",
			current: 3,
			targets: []int{1, 2},
			in:      "gitmap binary and otherpkg-v9 and otherpkg-v9",
			want:    "gitmap binary and otherpkg-v9 and otherpkg-v9",
		},
		{
			// v4: bare base preserved across full target sweep.
			name:    "v4_bare_base_preserved",
			current: 4,
			targets: []int{1, 2, 3},
			in:      "gitmap binary, otherpkg-v9, otherpkg-v9, otherpkg-v9",
			want:    "gitmap binary, otherpkg-v9, otherpkg-v9, otherpkg-v9",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := applyAllTargets(c.in, base, c.current, c.targets)
			if got != c.want {
				t.Fatalf("scope mismatch (current=%d).\n got:  %q\n want: %q", c.current, got, c.want)
			}
		})
	}
}
