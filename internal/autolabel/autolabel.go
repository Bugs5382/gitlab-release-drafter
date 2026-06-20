package autolabel

import (
	"regexp"
	"strings"

	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

// Match returns the labels the autolabeler rules imply for mr, excluding any
// labels the MR already carries. A rule fires if ANY of its branch regexes,
// title regexes, or file globs match.
func Match(mr model.MergeRequest, rules []config.Autolabel) []string {
	existing := map[string]struct{}{}
	for _, l := range mr.Labels {
		existing[l] = struct{}{}
	}

	var out []string
	added := map[string]struct{}{}
	add := func(label string) {
		if _, has := existing[label]; has {
			return
		}
		if _, dup := added[label]; dup {
			return
		}
		added[label] = struct{}{}
		out = append(out, label)
	}

	for _, r := range rules {
		if anyRegexMatch(r.Branch, mr.SourceBranch) ||
			anyRegexMatch(r.Title, mr.Title) ||
			anyGlobMatch(r.Files, mr.ChangedFiles) {
			add(r.Label)
		}
	}
	return out
}

func anyRegexMatch(patterns []string, s string) bool {
	if s == "" {
		return false
	}
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func anyGlobMatch(globs, files []string) bool {
	for _, g := range globs {
		re, err := regexp.Compile(globToRegex(g))
		if err != nil {
			continue
		}
		for _, f := range files {
			if re.MatchString(f) {
				return true
			}
		}
	}
	return false
}

// globToRegex converts a path glob (supporting **, *, ?) to an anchored regex.
func globToRegex(glob string) string {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); {
		c := glob[i]
		switch c {
		case '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				if i+2 < len(glob) && glob[i+2] == '/' {
					b.WriteString("(?:.*/)?")
					i += 3
					continue
				}
				b.WriteString(".*")
				i += 2
				continue
			}
			b.WriteString("[^/]*")
			i++
		case '?':
			b.WriteString("[^/]")
			i++
		default:
			if strings.IndexByte(`.+()|[]{}^$\`, c) >= 0 {
				b.WriteByte('\\')
			}
			b.WriteByte(c)
			i++
		}
	}
	b.WriteString("$")
	return b.String()
}
