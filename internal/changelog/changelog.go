package changelog

import (
	"regexp"
	"strings"
)

// Section is a Keep a Changelog subsection (e.g. "Added") with its bullet items.
type Section struct {
	Title string
	Items []string
}

// PromoteParams describes a new versioned section to insert.
type PromoteParams struct {
	Version     string // e.g. "1.2.0"
	Date        string // e.g. "2026-06-15"
	TagName     string // e.g. "v1.2.0"
	PreviousTag string // e.g. "v1.1.0" (may be empty)
	CompareURL  string // template with {from} and {to}; empty disables link maintenance
	Sections    []Section
}

const skeletonHeader = `# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).`

// Skeleton returns a canonical empty Keep a Changelog document.
func Skeleton() string {
	return skeletonHeader + "\n\n## [Unreleased]\n"
}

// RenderSections renders KAC subsections as "### Title\n\n- item" blocks.
func RenderSections(sections []Section) string {
	var blocks []string
	for _, s := range sections {
		if len(s.Items) == 0 {
			continue
		}
		var b strings.Builder
		b.WriteString("### " + s.Title + "\n")
		for _, it := range s.Items {
			b.WriteString("\n- " + it)
		}
		blocks = append(blocks, b.String())
	}
	return strings.Join(blocks, "\n\n")
}

// RenderUnreleased renders what the Unreleased section body would contain.
func RenderUnreleased(sections []Section) string {
	return RenderSections(sections)
}

var linkLineRe = regexp.MustCompile(`^\[[^\]]+\]:\s`)

// Promote inserts a new versioned section beneath the Unreleased heading,
// resets Unreleased, and (when CompareURL is set) maintains the link block.
func Promote(existing string, p PromoteParams) (string, error) {
	base := existing
	if strings.TrimSpace(base) == "" {
		base = Skeleton()
	}

	lines := strings.Split(strings.ReplaceAll(base, "\r\n", "\n"), "\n")

	unrelIdx := indexPrefix(lines, "## [Unreleased]")
	if unrelIdx < 0 {
		// Existing file lacks an Unreleased heading; rebuild from skeleton head.
		lines = strings.Split(Skeleton(), "\n")
		unrelIdx = indexPrefix(lines, "## [Unreleased]")
	}

	linkStart := linkBlockStart(lines)
	bodyEnd := linkStart
	if bodyEnd < 0 {
		bodyEnd = len(lines)
	}

	// Old version sections begin at the first "## [" after Unreleased.
	firstVersion := -1
	for i := unrelIdx + 1; i < bodyEnd; i++ {
		if strings.HasPrefix(lines[i], "## [") {
			firstVersion = i
			break
		}
	}
	var oldVersions []string
	if firstVersion >= 0 {
		oldVersions = trimBlank(lines[firstVersion:bodyEnd])
	}

	header := trimBlank(lines[:unrelIdx])
	var linkLines []string
	if linkStart >= 0 {
		linkLines = trimBlank(lines[linkStart:])
	}

	newSection := "## [" + p.Version + "] - " + p.Date
	if body := RenderSections(p.Sections); body != "" {
		newSection += "\n\n" + body
	}

	var out []string
	out = append(out, header...)
	out = append(out, "", "## [Unreleased]", "", newSection)
	if len(oldVersions) > 0 {
		out = append(out, "")
		out = append(out, oldVersions...)
	}

	newLinks := rebuildLinks(linkLines, p)
	if len(newLinks) > 0 {
		out = append(out, "")
		out = append(out, newLinks...)
	}

	return strings.Join(out, "\n") + "\n", nil
}

func rebuildLinks(existing []string, p PromoteParams) []string {
	if p.CompareURL == "" {
		// Cannot recompute; preserve existing version links, drop stale Unreleased.
		var kept []string
		for _, l := range existing {
			if strings.HasPrefix(l, "[Unreleased]:") {
				continue
			}
			kept = append(kept, l)
		}
		return kept
	}
	compare := func(from, to string) string {
		return strings.NewReplacer("{from}", from, "{to}", to).Replace(p.CompareURL)
	}
	var out []string
	out = append(out, "[Unreleased]: "+compare(p.TagName, "HEAD"))
	from := p.PreviousTag
	if from == "" {
		from = p.TagName
	}
	out = append(out, "["+p.Version+"]: "+compare(from, p.TagName))
	for _, l := range existing {
		if strings.HasPrefix(l, "[Unreleased]:") {
			continue
		}
		if strings.HasPrefix(l, "["+p.Version+"]:") {
			continue
		}
		out = append(out, l)
	}
	return out
}

func indexPrefix(lines []string, prefix string) int {
	for i, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return i
		}
	}
	return -1
}

// linkBlockStart returns the index where the trailing reference-link block
// begins, or -1 if there is none.
func linkBlockStart(lines []string) int {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	if end == 0 {
		return -1
	}
	start := end
	for start > 0 && linkLineRe.MatchString(lines[start-1]) {
		start--
	}
	if start == end {
		return -1
	}
	return start
}

func trimBlank(lines []string) []string {
	start, end := 0, len(lines)
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}
