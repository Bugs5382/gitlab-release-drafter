package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

// Semver is a simple major.minor.patch version.
type Semver struct {
	Major, Minor, Patch int
}

func (v Semver) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Less reports whether v < o.
func (v Semver) Less(o Semver) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor < o.Minor
	}
	return v.Patch < o.Patch
}

var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

// Parse parses "vX.Y.Z" or "X.Y.Z".
func Parse(s string) (Semver, error) {
	m := semverRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return Semver{}, fmt.Errorf("version: %q is not a semver", s)
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	return Semver{maj, min, pat}, nil
}

// Increment is a semver bump level.
type Increment int

const (
	Patch Increment = iota
	Minor
	Major
)

func (i Increment) String() string {
	switch i {
	case Major:
		return "major"
	case Minor:
		return "minor"
	default:
		return "patch"
	}
}

func incrementFromName(name string) Increment {
	switch name {
	case "major":
		return Major
	case "minor":
		return Minor
	default:
		return Patch
	}
}

// Base returns the highest semver among tags matching tagTemplate's prefix/suffix,
// or initial when none match.
func Base(tags []string, tagTemplate, initial string) Semver {
	prefix, suffix := tagAffixes(tagTemplate)
	var best Semver
	found := false
	for _, t := range tags {
		core := t
		if !strings.HasPrefix(core, prefix) || !strings.HasSuffix(core, suffix) {
			continue
		}
		core = strings.TrimSuffix(strings.TrimPrefix(core, prefix), suffix)
		v, err := Parse(core)
		if err != nil {
			continue
		}
		if !found || best.Less(v) {
			best, found = v, true
		}
	}
	if found {
		return best
	}
	v, err := Parse(initial)
	if err != nil {
		return Semver{}
	}
	return v
}

func tagAffixes(tmpl string) (prefix, suffix string) {
	idx := strings.Index(tmpl, "{version}")
	if idx < 0 {
		return "", ""
	}
	return tmpl[:idx], tmpl[idx+len("{version}"):]
}

// Resolve returns the highest increment implied by the MRs' labels.
// MRs whose labels match no increment use defaultIncrement.
func Resolve(mrs []model.MergeRequest, increments map[string][]string, defaultIncrement string) Increment {
	labelToInc := map[string]Increment{}
	for name, labels := range increments {
		inc := incrementFromName(name)
		for _, l := range labels {
			labelToInc[l] = inc
		}
	}
	def := incrementFromName(defaultIncrement)

	best := Patch
	any := false
	for _, mr := range mrs {
		any = true
		mrInc := def
		matched := false
		for _, l := range mr.Labels {
			if inc, ok := labelToInc[l]; ok {
				if !matched || inc > mrInc {
					mrInc = inc
				}
				matched = true
			}
		}
		if mrInc > best {
			best = mrInc
		}
	}
	if !any {
		return def
	}
	return best
}

// Bump returns base incremented by inc.
func Bump(base Semver, inc Increment) Semver {
	switch inc {
	case Major:
		return Semver{base.Major + 1, 0, 0}
	case Minor:
		return Semver{base.Major, base.Minor + 1, 0}
	default:
		return Semver{base.Major, base.Minor, base.Patch + 1}
	}
}

// RenderTag substitutes version placeholders in a template.
func RenderTag(tmpl string, v Semver) string {
	r := strings.NewReplacer(
		"{version}", v.String(),
		"{major}", strconv.Itoa(v.Major),
		"{minor}", strconv.Itoa(v.Minor),
		"{patch}", strconv.Itoa(v.Patch),
	)
	return r.Replace(tmpl)
}
